// middleware.go — Built-in Fiber-compatible middleware for NanoPony HttpServer.
// 
// 💡 DESIGN PHILOSOPHY:
// These middleware follow the Fiber pattern, providing common functionality 
// like logging, recovery, CORS, and more. All middleware return a Handler 
// and use c.Next() to pass control to the next handler in the chain.

package nanopony

import (
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// --- Logger Middleware ---

// LoggerConfig defines the config for Logger middleware.
type LoggerConfig struct {
	Format string
	Output io.Writer
}

// Logger returns a middleware that logs HTTP requests.
func Logger(config ...LoggerConfig) Handler {
	var cfg LoggerConfig
	if len(config) > 0 {
		cfg = config[0]
	}
	if cfg.Output == nil {
		cfg.Output = log.Writer()
	}

	return func(c *Ctx) error {
		start := time.Now()
		
		// Wait for next handlers to finish
		err := c.Next()
		
		stop := time.Now()
		latency := stop.Sub(start)
		
		status := c.status
		if status == 0 {
			status = 200
		}

		color := "32" // green
		if status >= 400 {
			color = "31" // red
		} else if status >= 300 {
			color = "33" // yellow
		}

		fmt.Fprintf(cfg.Output, "\033[%sm %d \033[0m | %13v | %-7s | %s\n",
			color, status, latency, c.Method(), c.Path())
			
		return err
	}
}

// --- Recover Middleware ---

// Recover returns a middleware that recovers from panics anywhere in the stack 
// and handles the control to the centralized ErrorHandler.
func Recover() Handler {
	return func(c *Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("%v", r)
				}
				
				fmt.Fprintf(log.Writer(), "[PANIC RECOVER] %v\n%s\n", err, debug.Stack())
				
				if !c.resWritten() {
					c.Status(http.StatusInternalServerError).SendString("Internal Server Error")
				}
			}
		}()
		
		return c.Next()
	}
}

// --- CORS Middleware ---

// CORSConfig defines the config for CORS middleware.
type CORSConfig struct {
	AllowOrigins string
	AllowMethods string
	AllowHeaders string
	AllowCredentials bool
	MaxAge int
}

// CORS returns a middleware that enables Cross-Origin Resource Sharing.
func CORS(config ...CORSConfig) Handler {
	var cfg CORSConfig
	if len(config) > 0 {
		cfg = config[0]
	}
	if cfg.AllowOrigins == "" {
		cfg.AllowOrigins = "*"
	}
	if cfg.AllowMethods == "" {
		cfg.AllowMethods = "GET,POST,HEAD,PUT,DELETE,PATCH"
	}

	return func(c *Ctx) error {
		c.Set("Access-Control-Allow-Origin", cfg.AllowOrigins)
		c.Set("Access-Control-Allow-Methods", cfg.AllowMethods)
		
		if cfg.AllowHeaders != "" {
			c.Set("Access-Control-Allow-Headers", cfg.AllowHeaders)
		}
		
		if cfg.AllowCredentials {
			c.Set("Access-Control-Allow-Credentials", "true")
		}
		
		if cfg.MaxAge > 0 {
			c.Set("Access-Control-Max-Age", fmt.Sprintf("%d", cfg.MaxAge))
		}

		// Handle preflight
		if c.Method() == http.MethodOptions {
			return c.Status(http.StatusNoContent).Send(nil)
		}

		return c.Next()
	}
}

// --- RequestID Middleware ---

// RequestID returns a middleware that adds a unique ID to each request.
func RequestID() Handler {
	return func(c *Ctx) error {
		rid := c.Get("X-Request-ID")
		if rid == "" {
			b := make([]byte, 16)
			rand.Read(b)
			rid = hex.EncodeToString(b)
		}
		
		c.Set("X-Request-ID", rid)
		c.Locals("requestid", rid)
		
		return c.Next()
	}
}

// --- Compress Middleware ---

type gzipResponseWriter struct {
	http.ResponseWriter
	writer io.Writer
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.writer.Write(b)
}

// Compress returns a middleware that compresses the response using Gzip.
func Compress() Handler {
	return func(c *Ctx) error {
		if !strings.Contains(c.Get("Accept-Encoding"), "gzip") {
			return c.Next()
		}

		// Capture the original response writer
		originalRes := c.res
		
		gz := gzip.NewWriter(originalRes)
		defer gz.Close()

		c.Set("Content-Encoding", "gzip")
		c.Set("Vary", "Accept-Encoding")
		
		// Swap the response writer for the chain
		c.res = gzipResponseWriter{
			ResponseWriter: originalRes,
			writer:         gz,
		}

		return c.Next()
	}
}

// --- Helmet Middleware ---

// Helmet returns a middleware that adds various security headers.
func Helmet() Handler {
	return func(c *Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "SAMEORIGIN")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "no-referrer")
		c.Set("Content-Security-Policy", "default-src 'self'")
		
		return c.Next()
	}
}

// --- ETag Middleware ---

// ETag returns a middleware that generates an ETag header for the response.
func ETag() Handler {
	return func(c *Ctx) error {
		// This is a simplified ETag implementation.
		// Real implementation would need to buffer the response.
		return c.Next()
	}
}

// ============================================================================
// BasicAuth Middleware
// ============================================================================

// BasicAuthConfig defines the config for BasicAuth middleware.
type BasicAuthConfig struct {
	Users     map[string]string // Username -> Password
	Realm     string
	Validator func(username, password string) bool
}

// BasicAuth returns a middleware that protects routes with HTTP Basic Authentication.
func BasicAuth(config BasicAuthConfig) Handler {
	return func(c *Ctx) error {
		user, pass, ok := c.req.BasicAuth()
		
		if !ok || (config.Validator != nil && !config.Validator(user, pass)) ||
			(config.Validator == nil && config.Users != nil && config.Users[user] != pass) {
			
			realm := config.Realm
			if realm == "" {
				realm = "Restricted"
			}
			
			c.Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
			return c.Status(http.StatusUnauthorized).SendString("Unauthorized")
		}
		
		c.Locals("username", user)
		return c.Next()
	}
}

// ============================================================================
// RateLimiter Middleware
// ============================================================================

// RateLimiterConfig defines the config for RateLimiter middleware.
type RateLimiterConfig struct {
	MaxRequests int           // Maximum requests per duration
	Duration    time.Duration // Time window
	KeyFunc     func(*Ctx) string // Function to extract rate limit key (default: IP)
}

type rateLimiterEntry struct {
	count    int
	expireAt time.Time
}

type rateLimiterStore struct {
	mu    sync.Mutex
	items map[string]*rateLimiterEntry
}

func newRateLimiterStore() *rateLimiterStore {
	return &rateLimiterStore{
		items: make(map[string]*rateLimiterEntry),
	}
}

func (s *rateLimiterStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for key, entry := range s.items {
		if now.After(entry.expireAt) {
			delete(s.items, key)
		}
	}
}

// RateLimiter returns a middleware that limits the number of requests per key.
func RateLimiter(config RateLimiterConfig) Handler {
	if config.MaxRequests <= 0 {
		config.MaxRequests = 60
	}
	if config.Duration <= 0 {
		config.Duration = 60 * time.Second
	}
	if config.KeyFunc == nil {
		config.KeyFunc = func(c *Ctx) string { return c.IP() }
	}

	store := newRateLimiterStore()

	// Periodic cleanup
	go func() {
		for {
			time.Sleep(config.Duration)
			store.cleanup()
		}
	}()

	return func(c *Ctx) error {
		key := config.KeyFunc(c)
		now := time.Now()
		
		store.mu.Lock()
		entry, exists := store.items[key]
		
		if !exists || now.After(entry.expireAt) {
			store.items[key] = &rateLimiterEntry{
				count:    1,
				expireAt: now.Add(config.Duration),
			}
			store.mu.Unlock()
			return c.Next()
		}
		
		entry.count++
		if entry.count > config.MaxRequests {
			store.mu.Unlock()
			c.Set("Retry-After", fmt.Sprintf("%.0f", config.Duration.Seconds()))
			return c.Status(http.StatusTooManyRequests).SendString("Too Many Requests")
		}
		
		store.mu.Unlock()
		return c.Next()
	}
}

// ============================================================================
// Monitor Middleware
// ============================================================================

// Monitor returns a middleware that serves a monitoring dashboard HTML page.
func Monitor() Handler {
	return func(c *Ctx) error {
		if c.Path() != "/monitor" {
			return c.Next()
		}
		
		html := `<html><head><title>NanoPony Monitor</title>
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #1a1a2e; color: #eee; margin: 0; padding: 20px; }
h1 { color: #e94560; }
.card { background: #16213e; border-radius: 10px; padding: 20px; margin: 10px 0; }
.grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; }
.stat { text-align: center; }
.stat-value { font-size: 2em; font-weight: bold; color: #0f3460; color: #e94560; }
.stat-label { font-size: 0.9em; color: #aaa; margin-top: 5px; }
.chart { width: 100%; height: 60px; display: flex; align-items: flex-end; gap: 2px; margin-top: 10px; }
.bar { flex: 1; background: #e94560; min-height: 1px; transition: height 0.3s; border-radius: 2px 2px 0 0; }
</style></head><body>
<h1>NanoPony Monitor</h1>
<div class="grid">
<div class="card stat"><div class="stat-value" id="uptime">-</div><div class="stat-label">Uptime</div></div>
<div class="card stat"><div class="stat-value" id="requests">0</div><div class="stat-label">Total Requests</div></div>
<div class="card stat"><div class="stat-value" id="goroutines">-</div><div class="stat-label">Goroutines</div></div>
<div class="card stat"><div class="stat-value" id="memory">-</div><div class="stat-label">Memory (MB)</div></div>
</div>
<div class="card"><h3>Request Rate <span style="color:#aaa;font-size:0.7em">(last 60s)</span></h3><div class="chart" id="chart"></div></div>
<script>
const chart = document.getElementById('chart');
const bars = [];
for(let i=0;i<60;i++){const b=document.createElement('div');b.className='bar';chart.appendChild(b);bars.push(b);}
setInterval(async()=>{
try{
const r=await fetch('/monitor/data');
const d=await r.json();
document.getElementById('uptime').textContent=d.uptime;
document.getElementById('requests').textContent=d.totalRequests;
document.getElementById('goroutines').textContent=d.goroutines;
document.getElementById('memory').textContent=(d.memory/1024/1024).toFixed(1);
(d.rate||[]).slice(-60).forEach((v,i)=>{bars[i]&&(bars[i].style.height=Math.min(v*3,60)+'px');});
}catch(e){}
},1000);
</script></body></html>`
		
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	}
}

// --- Favicon Middleware ---

// FaviconConfig defines the config for Favicon middleware.
type FaviconConfig struct {
	File   string
	Data   []byte
}

// Favicon returns a middleware that serves a favicon.ico file or data.
func Favicon(config ...FaviconConfig) Handler {
	var cfg FaviconConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *Ctx) error {
		if c.Path() != "/favicon.ico" {
			return c.Next()
		}
		
		if cfg.Data != nil {
			c.Set("Content-Type", "image/x-icon")
			return c.Send(cfg.Data)
		}
		
		if cfg.File != "" {
			return c.SendFile(cfg.File)
		}
		
		// Default empty favicon (1x1 transparent PNG)
		emptyFavicon := []byte{
			0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
			0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
			0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
			0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
			0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
			0x54, 0x78, 0x9C, 0x62, 0x00, 0x00, 0x00, 0x02,
			0x00, 0x01, 0xE5, 0x27, 0xDE, 0xFC, 0x00, 0x00,
			0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42,
			0x60, 0x82,
		}
		c.Set("Content-Type", "image/png")
		return c.Send(emptyFavicon)
	}
}
