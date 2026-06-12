// http_server.go — Fiber-compatible HTTP framework built on net/http.
// 
// 💡 DESIGN PHILOSOPHY:
// This module provides an API surface identical to the Fiber web framework,
// allowing developers to use familiar patterns (app.Get, c.JSON, c.Next, etc.)
// while being built on the standard net/http package for maximum stability
// and compatibility.
//
// Key Features:
// 1. Fiber-like Router: Supports static routes, parameters (:id), and wildcards (*).
// 2. Middleware Chain: Support for global, group, and route-specific middleware.
// 3. Fiber-compatible Context: Ctx struct with all standard Fiber methods.
// 4. Group Routing: Modular route organization.
// 5. Integration: Seamlessly integrates into the NanoPony builder pattern.

package nanopony

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Handler is the signature for HTTP request handlers.
type Handler func(c *Ctx) error

// Map is a shortcut for map[string]interface{}.
type Map map[string]interface{}

// HttpServer is the main application struct.
type HttpServer struct {
	router      *router
	middleware  []*middlewareItem
	groups      []*Group
	server      *http.Server
	config      HttpConfig
	logger      *LogManager
	mu          sync.RWMutex
	startupTime time.Time
	shutdown    bool
}

// HttpConfig holds server configuration.
type HttpConfig struct {
	Port           string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	AppName        string
	ServerHeader   string
	CaseSensitive  bool
	StrictRouting  bool
	DisableKeepAlive bool
	BodyLimit      int
}

type middlewareItem struct {
	prefix   string
	handlers []Handler
}

// Ctx represents the context of an HTTP request.
type Ctx struct {
	app      *HttpServer
	req      *http.Request
	res      http.ResponseWriter
	path     string
	method   string
	params   map[string]string
	query    url.Values
	body     []byte
	status   int
	index    int
	handlers []Handler
	locals   map[string]interface{}
}

// Cookie represents an HTTP cookie.
type Cookie struct {
	Name     string
	Value    string
	Path     string
	Domain   string
	Expires  time.Time
	Secure   bool
	HTTPOnly bool
	SameSite string
}

// NewHttpServer creates a new server instance.
func NewHttpServer(cfg ...HttpConfig) *HttpServer {
	var config HttpConfig
	if len(cfg) > 0 {
		config = cfg[0]
	}

	if config.Port == "" {
		config.Port = "3000"
	}

	return &HttpServer{
		router:      newRouter(),
		middleware:  make([]*middlewareItem, 0),
		groups:      make([]*Group, 0),
		config:      config,
		startupTime: time.Now(),
	}
}

// --- HttpServer Methods (Fiber-compatible API) ---

func (app *HttpServer) Get(path string, handlers ...Handler) *HttpServer {
	return app.Add(http.MethodGet, path, handlers...)
}

func (app *HttpServer) Post(path string, handlers ...Handler) *HttpServer {
	return app.Add(http.MethodPost, path, handlers...)
}

func (app *HttpServer) Put(path string, handlers ...Handler) *HttpServer {
	return app.Add(http.MethodPut, path, handlers...)
}

func (app *HttpServer) Delete(path string, handlers ...Handler) *HttpServer {
	return app.Add(http.MethodDelete, path, handlers...)
}

func (app *HttpServer) Patch(path string, handlers ...Handler) *HttpServer {
	return app.Add(http.MethodPatch, path, handlers...)
}

func (app *HttpServer) Head(path string, handlers ...Handler) *HttpServer {
	return app.Add(http.MethodHead, path, handlers...)
}

func (app *HttpServer) Options(path string, handlers ...Handler) *HttpServer {
	return app.Add(http.MethodOptions, path, handlers...)
}

func (app *HttpServer) Trace(path string, handlers ...Handler) *HttpServer {
	return app.Add(http.MethodTrace, path, handlers...)
}

func (app *HttpServer) Connect(path string, handlers ...Handler) *HttpServer {
	return app.Add(http.MethodConnect, path, handlers...)
}

func (app *HttpServer) All(path string, handlers ...Handler) *HttpServer {
	methods := []string{
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
		http.MethodPatch, http.MethodHead, http.MethodOptions,
	}
	for _, m := range methods {
		app.Add(m, path, handlers...)
	}
	return app
}

func (app *HttpServer) Add(method, path string, handlers ...Handler) *HttpServer {
	app.router.add(method, path, handlers...)
	return app
}

func (app *HttpServer) Use(args ...interface{}) *HttpServer {
	var prefix string = ""
	var handlers []Handler

	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			prefix = v
		case Handler:
			handlers = append(handlers, v)
		case func(*Ctx) error:
			handlers = append(handlers, v)
		}
	}

	if len(handlers) > 0 {
		app.middleware = append(app.middleware, &middlewareItem{
			prefix:   prefix,
			handlers: handlers,
		})
	}

	return app
}

func (app *HttpServer) Group(prefix string, handlers ...Handler) *Group {
	group := &Group{
		prefix:     prefix,
		app:        app,
		middleware: handlers,
	}
	app.groups = append(app.groups, group)
	return group
}

func (app *HttpServer) Static(prefix, root string) *HttpServer {
	app.Get(strings.TrimSuffix(prefix, "/")+"/*", func(c *Ctx) error {
		path := c.Params("*")
		if path == "" {
			path = "index.html"
		}
		fullPath := filepath.Join(root, path)
		return c.SendFile(fullPath)
	})
	return app
}

func (app *HttpServer) Listen(addr string) error {
	if !strings.Contains(addr, ":") {
		addr = ":" + addr
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:         addr,
		Handler:      app,
		ReadTimeout:  app.config.ReadTimeout,
		WriteTimeout: app.config.WriteTimeout,
		IdleTimeout:  app.config.IdleTimeout,
	}

	if app.logger != nil {
		app.logger.LogFramework("INFO", "HttpServer", fmt.Sprintf("Server starting on %s", addr))
	}

	app.mu.Lock()
	if app.shutdown {
		app.mu.Unlock()
		_ = ln.Close()
		return http.ErrServerClosed
	}
	app.server = srv
	app.mu.Unlock()

	return srv.Serve(ln)
}

func (app *HttpServer) Shutdown() error {
	app.mu.Lock()
	app.shutdown = true
	srv := app.server
	app.mu.Unlock()

	if srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}
	return nil
}

func (app *HttpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	method := r.Method

	// Match route
	handlers, params := app.router.match(method, path)

	// Collect middleware
	chain := make([]Handler, 0)
	for _, m := range app.middleware {
		if strings.HasPrefix(path, m.prefix) {
			chain = append(chain, m.handlers...)
		}
	}

	// Add route handlers
	if len(handlers) > 0 {
		chain = append(chain, handlers...)
	} else {
		// 404 handler
		chain = append(chain, func(c *Ctx) error {
			return c.Status(404).SendString("Not Found")
		})
	}

	// Create context
	c := &Ctx{
		app:      app,
		req:      r,
		res:      w,
		path:     path,
		method:   method,
		params:   params,
		handlers: chain,
		index:    -1,
		locals:   make(map[string]interface{}),
	}

	if app.config.ServerHeader != "" {
		c.Set("Server", app.config.ServerHeader)
	}

	// Execute chain
	if err := c.Next(); err != nil {
		if app.logger != nil {
			app.logger.LogFramework("ERROR", "HttpServer", err.Error())
		}
		if !c.resWritten() {
			c.Status(500).SendString("Internal Server Error")
		}
	}
}

func (c *Ctx) resWritten() bool {
	return c.status != 0
}

// --- Ctx Methods (Fiber-compatible API) ---

func (c *Ctx) Next() error {
	c.index++
	if c.index < len(c.handlers) {
		return c.handlers[c.index](c)
	}
	return nil
}

func (c *Ctx) Status(status int) *Ctx {
	c.status = status
	c.res.WriteHeader(status)
	return c
}

func (c *Ctx) Send(body []byte) error {
	if c.status == 0 {
		c.Status(200)
	}
	_, err := c.res.Write(body)
	return err
}

func (c *Ctx) SendString(body string) error {
	c.Set("Content-Type", "text/plain; charset=utf-8")
	return c.Send([]byte(body))
}

func (c *Ctx) JSON(data interface{}) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	c.Set("Content-Type", "application/json")
	return c.Send(raw)
}

func (c *Ctx) Body() []byte {
	if c.body == nil && c.req.Body != nil {
		body, _ := io.ReadAll(c.req.Body)
		c.body = body
		c.req.Body = io.NopCloser(bytes.NewBuffer(body))
	}
	return c.body
}

func (c *Ctx) BodyParser(out interface{}) error {
	body := c.Body()
	contentType := c.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		return json.Unmarshal(body, out)
	}
	
	// Basic support for form values could be added here
	return errors.New("unsupported content type for BodyParser")
}

func (c *Ctx) Params(key string, defaultVal ...string) string {
	if val, ok := c.params[key]; ok {
		return val
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return ""
}

func (c *Ctx) Query(key string, defaultVal ...string) string {
	if c.query == nil {
		c.query = c.req.URL.Query()
	}
	val := c.query.Get(key)
	if val == "" && len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return val
}

func (c *Ctx) Set(key, value string) {
	c.res.Header().Set(key, value)
}

func (c *Ctx) Get(key string) string {
	return c.req.Header.Get(key)
}

func (c *Ctx) Method() string {
	return c.method
}

func (c *Ctx) Path() string {
	return c.path
}

func (c *Ctx) Hostname() string {
	return c.req.Host
}

func (c *Ctx) IP() string {
	ip, _, _ := net.SplitHostPort(c.req.RemoteAddr)
	return ip
}

func (c *Ctx) Locals(key string, value ...interface{}) interface{} {
	if len(value) > 0 {
		c.locals[key] = value[0]
		return value[0]
	}
	return c.locals[key]
}

func (c *Ctx) Redirect(location string, status ...int) error {
	s := http.StatusFound
	if len(status) > 0 {
		s = status[0]
	}
	http.Redirect(c.res, c.req, location, s)
	c.status = s
	return nil
}

func (c *Ctx) SendFile(path string) error {
	http.ServeFile(c.res, c.req, path)
	c.status = 200
	return nil
}

func (c *Ctx) FormValue(key string) string {
	return c.req.FormValue(key)
}

func (c *Ctx) Type(ext string) *Ctx {
	if !strings.Contains(ext, "/") {
		ext = mime.TypeByExtension("." + ext)
	}
	c.Set("Content-Type", ext)
	return c
}

func (c *Ctx) Cookie(cookie *Cookie) {
	http.SetCookie(c.res, &http.Cookie{
		Name:     cookie.Name,
		Value:    cookie.Value,
		Path:     cookie.Path,
		Domain:   cookie.Domain,
		Expires:  cookie.Expires,
		Secure:   cookie.Secure,
		HttpOnly: cookie.HTTPOnly,
	})
}

func (c *Ctx) MultipartForm() (*multipart.Form, error) {
	err := c.req.ParseMultipartForm(32 << 20) // 32MB default
	return c.req.MultipartForm, err
}

// --- Group Struct ---

type Group struct {
	prefix     string
	app        *HttpServer
	middleware []Handler
}

func (g *Group) Get(path string, handlers ...Handler) *Group { return g.Add(http.MethodGet, path, handlers...) }
func (g *Group) Post(path string, handlers ...Handler) *Group { return g.Add(http.MethodPost, path, handlers...) }
func (g *Group) Put(path string, handlers ...Handler) *Group { return g.Add(http.MethodPut, path, handlers...) }
func (g *Group) Delete(path string, handlers ...Handler) *Group { return g.Add(http.MethodDelete, path, handlers...) }

func (g *Group) Add(method, path string, handlers ...Handler) *Group {
	fullPath := g.prefix + path
	fullHandlers := append(g.middleware, handlers...)
	g.app.Add(method, fullPath, fullHandlers...)
	return g
}

func (g *Group) Use(handlers ...Handler) *Group {
	g.middleware = append(g.middleware, handlers...)
	return g
}

func (g *Group) Group(prefix string, handlers ...Handler) *Group {
	newPrefix := g.prefix + prefix
	newHandlers := append(g.middleware, handlers...)
	return g.app.Group(newPrefix, newHandlers...)
}

// --- Router implementation ---

type router struct {
	root *node
}

type nodeType uint8

const (
	nodeStatic nodeType = iota
	nodeParam
	nodeWildcard
)

type node struct {
	segment  string
	ntype    nodeType
	children []*node
	handlers map[string][]Handler
}

func newRouter() *router {
	return &router{
		root: &node{
			segment:  "/",
			children: make([]*node, 0),
			handlers: make(map[string][]Handler),
		},
	}
}

func (r *router) add(method, path string, handlers ...Handler) {
	if path == "" || path == "/" {
		r.root.handlers[method] = handlers
		return
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	current := r.root

	for _, part := range parts {
		var ntype nodeType = nodeStatic
		segment := part

		if strings.HasPrefix(part, ":") {
			ntype = nodeParam
			segment = part[1:]
		} else if part == "*" {
			ntype = nodeWildcard
			segment = "*"
		}

		// Find child
		var child *node
		for _, c := range current.children {
			if c.segment == segment && c.ntype == ntype {
				child = c
				break
			}
		}

		if child == nil {
			child = &node{
				segment:  segment,
				ntype:    ntype,
				children: make([]*node, 0),
				handlers: make(map[string][]Handler),
			}
			current.children = append(current.children, child)
		}
		current = child
	}

	current.handlers[method] = handlers
}

func (r *router) match(method, path string) ([]Handler, map[string]string) {
	if path == "/" || path == "" {
		return r.root.handlers[method], nil
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	params := make(map[string]string)
	
	handlers := r.matchRecursive(r.root, method, parts, params)
	return handlers, params
}

func (r *router) matchRecursive(n *node, method string, parts []string, params map[string]string) []Handler {
	if len(parts) == 0 {
		return n.handlers[method]
	}

	segment := parts[0]

	// 1. Try static match
	for _, child := range n.children {
		if child.ntype == nodeStatic && child.segment == segment {
			h := r.matchRecursive(child, method, parts[1:], params)
			if h != nil {
				return h
			}
		}
	}

	// 2. Try param match
	for _, child := range n.children {
		if child.ntype == nodeParam {
			h := r.matchRecursive(child, method, parts[1:], params)
			if h != nil {
				params[child.segment] = segment
				return h
			}
		}
	}

	// 3. Try wildcard match
	for _, child := range n.children {
		if child.ntype == nodeWildcard {
			params["*"] = strings.Join(parts, "/")
			return child.handlers[method]
		}
	}

	return nil
}

// WithLogger sets the logger for the HttpServer.
func (app *HttpServer) WithLogger(lm *LogManager) *HttpServer {
	app.logger = lm
	return app
}
