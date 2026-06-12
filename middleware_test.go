package nanopony

import (
	"net/http/httptest"
	"time"
	"testing"
)

func TestMiddlewareLogger(t *testing.T) {
	app := NewHttpServer()
	app.Use(Logger())
	
	app.Get("/test", func(c *Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestMiddlewareRecover(t *testing.T) {
	app := NewHttpServer()
	app.Use(Recover())
	
	app.Get("/panic", func(c *Ctx) error {
		panic("something went wrong")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	
	// We want to make sure the app doesn't crash
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("App panicked even with Recover middleware")
		}
	}()
	
	app.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("Expected 500, got %d", w.Code)
	}
}

func TestMiddlewareCORS(t *testing.T) {
	app := NewHttpServer()
	app.Use(CORS(CORSConfig{
		AllowOrigins: "http://example.com",
	}))
	
	app.Get("/cors", func(c *Ctx) error {
		return c.SendString("cors ok")
	})

	req := httptest.NewRequest("GET", "/cors", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("CORS header not set correctly")
	}
}

func TestMiddlewareRequestID(t *testing.T) {
	app := NewHttpServer()
	app.Use(RequestID())
	
	app.Get("/rid", func(c *Ctx) error {
		return c.SendString(c.Get("X-Request-ID"))
	})

	req := httptest.NewRequest("GET", "/rid", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") == "" {
		t.Errorf("Request ID not generated")
	}
}

func TestMiddlewareHelmet(t *testing.T) {
	app := NewHttpServer()
	app.Use(Helmet())
	
	app.Get("/security", func(c *Ctx) error {
		return c.SendString("secure")
	})

	req := httptest.NewRequest("GET", "/security", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Errorf("Security headers not set")
	}
}

func TestMiddlewareBasicAuth(t *testing.T) {
	app := NewHttpServer()
	app.Use(BasicAuth(BasicAuthConfig{
		Users: map[string]string{"admin": "1234"},
	}))
	
	app.Get("/secret", func(c *Ctx) error {
		return c.SendString("secret data")
	})

	// No auth
	req := httptest.NewRequest("GET", "/secret", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Errorf("Expected 401, got %d", w.Code)
	}

	// Valid auth
	req = httptest.NewRequest("GET", "/secret", nil)
	req.SetBasicAuth("admin", "1234")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 200 || w.Body.String() != "secret data" {
		t.Errorf("Auth failed: %d, %s", w.Code, w.Body.String())
	}
}

func TestMiddlewareRateLimiter(t *testing.T) {
	app := NewHttpServer()
	app.Use(RateLimiter(RateLimiterConfig{
		MaxRequests: 2,
		Duration:    time.Second,
	}))
	
	app.Get("/limited", func(c *Ctx) error {
		return c.SendString("ok")
	})

	for i := 1; i <= 3; i++ {
		req := httptest.NewRequest("GET", "/limited", nil)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		
		if i <= 2 && w.Code != 200 {
			t.Errorf("Request %d failed unexpectedly: %d", i, w.Code)
		}
		if i == 3 && w.Code != 429 {
			t.Errorf("Request 3 should have been limited: %d", w.Code)
		}
	}
}
