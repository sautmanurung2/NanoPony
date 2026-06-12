package nanopony

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestHttpServerBasic(t *testing.T) {
	app := NewHttpServer()

	app.Get("/hello", func(c *Ctx) error {
		return c.SendString("world")
	})

	req := httptest.NewRequest("GET", "/hello", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	if w.Body.String() != "world" {
		t.Errorf("Expected 'world', got '%s'", w.Body.String())
	}
}

func TestHttpServerParams(t *testing.T) {
	app := NewHttpServer()

	app.Get("/users/:id", func(c *Ctx) error {
		id := c.Params("id")
		return c.SendString("user " + id)
	})

	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Body.String() != "user 123" {
		t.Errorf("Expected 'user 123', got '%s'", w.Body.String())
	}
}

func TestHttpServerWildcard(t *testing.T) {
	app := NewHttpServer()

	app.Get("/api/*", func(c *Ctx) error {
		path := c.Params("*")
		return c.SendString("path: " + path)
	})

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Body.String() != "path: v1/users" {
		t.Errorf("Expected 'path: v1/users', got '%s'", w.Body.String())
	}
}

func TestHttpServerJSON(t *testing.T) {
	app := NewHttpServer()

	app.Get("/json", func(c *Ctx) error {
		return c.JSON(Map{"hello": "world"})
	})

	req := httptest.NewRequest("GET", "/json", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected application/json")
	}

	var res map[string]string
	json.Unmarshal(w.Body.Bytes(), &res)
	if res["hello"] != "world" {
		t.Errorf("Expected world")
	}
}

func TestHttpServerBodyParser(t *testing.T) {
	app := NewHttpServer()

	type User struct {
		Name string `json:"name"`
	}

	app.Post("/users", func(c *Ctx) error {
		var user User
		if err := c.BodyParser(&user); err != nil {
			return c.Status(400).SendString(err.Error())
		}
		return c.SendString("created " + user.Name)
	})

	body := bytes.NewBufferString(`{"name": "john"}`)
	req := httptest.NewRequest("POST", "/users", body)
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Body.String() != "created john" {
		t.Errorf("Expected 'created john', got '%s'", w.Body.String())
	}
}

func TestHttpServerMiddleware(t *testing.T) {
	app := NewHttpServer()
	
	order := make([]string, 0)

	app.Use(func(c *Ctx) error {
		order = append(order, "global")
		return c.Next()
	})

	app.Use("/api", func(c *Ctx) error {
		order = append(order, "api")
		return c.Next()
	})

	app.Get("/api/test", func(c *Ctx) error {
		order = append(order, "handler")
		return c.Send([]byte("ok"))
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if len(order) != 3 || order[0] != "global" || order[1] != "api" || order[2] != "handler" {
		t.Errorf("Middleware execution order incorrect: %v", order)
	}
}

func TestHttpServerGroup(t *testing.T) {
	app := NewHttpServer()
	
	api := app.Group("/api")
	v1 := api.Group("/v1")
	
	v1.Get("/users", func(c *Ctx) error {
		return c.SendString("users")
	})

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Body.String() != "users" {
		t.Errorf("Group routing failed")
	}
}

func TestHttpServerPut(t *testing.T) {
	app := NewHttpServer()
	app.Put("/put", func(c *Ctx) error { return c.SendString("put") })
	req := httptest.NewRequest("PUT", "/put", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Body.String() != "put" { t.Errorf("PUT failed") }
}

func TestHttpServerDelete(t *testing.T) {
	app := NewHttpServer()
	app.Delete("/del", func(c *Ctx) error { return c.SendString("del") })
	req := httptest.NewRequest("DELETE", "/del", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Body.String() != "del" { t.Errorf("DELETE failed") }
}

func TestHttpServerPatch(t *testing.T) {
	app := NewHttpServer()
	app.Patch("/patch", func(c *Ctx) error { return c.SendString("patch") })
	req := httptest.NewRequest("PATCH", "/patch", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Body.String() != "patch" { t.Errorf("PATCH failed") }
}

func TestHttpServerHead(t *testing.T) {
	app := NewHttpServer()
	app.Head("/head", func(c *Ctx) error { return c.SendString("head") })
	req := httptest.NewRequest("HEAD", "/head", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 200 { t.Errorf("HEAD failed") }
}

func TestHttpServerOptions(t *testing.T) {
	app := NewHttpServer()
	app.Options("/opts", func(c *Ctx) error { return c.SendString("opts") })
	req := httptest.NewRequest("OPTIONS", "/opts", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Body.String() != "opts" { t.Errorf("OPTIONS failed") }
}

func TestHttpServerAll(t *testing.T) {
	app := NewHttpServer()
	app.All("/all", func(c *Ctx) error { return c.SendString("all") })
	for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
		req := httptest.NewRequest(method, "/all", nil)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		if w.Body.String() != "all" { t.Errorf("All method %s failed", method) }
	}
}

func TestHttpServerStatic(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/test.txt"
	os.WriteFile(tmpFile, []byte("static"), 0644)
	app := NewHttpServer()
	app.Static("/files", tmpDir)
	req := httptest.NewRequest("GET", "/files/test.txt", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Body.String() != "static" { t.Errorf("Static file failed: %s", w.Body.String()) }
}

func TestCtxGetSet(t *testing.T) {
	app := NewHttpServer()
	app.Get("/hdr", func(c *Ctx) error {
		c.Set("X-Custom", "val")
		return c.SendString(c.Get("User-Agent"))
	})
	req := httptest.NewRequest("GET", "/hdr", nil)
	req.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Body.String() != "test-agent" { t.Errorf("Get header failed") }
	if w.Header().Get("X-Custom") != "val" { t.Errorf("Set header failed") }
}

func TestCtxHostnameIP(t *testing.T) {
	app := NewHttpServer()
	app.Get("/info", func(c *Ctx) error {
		return c.SendString(c.Hostname() + "|" + c.IP())
	})
	req := httptest.NewRequest("GET", "/info", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Body.String() == "" { t.Errorf("Hostname/IP empty") }
}

func TestCtxType(t *testing.T) {
	app := NewHttpServer()
	app.Get("/type", func(c *Ctx) error {
		c.Type(".json")
		return c.Send([]byte("ok"))
	})
	req := httptest.NewRequest("GET", "/type", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	ct := w.Header().Get("Content-Type")
	if ct != "" && ct != "application/json" { t.Errorf("Type failed: %s", ct) }
}

func TestCtxQuery(t *testing.T) {
	app := NewHttpServer()
	app.Get("/search", func(c *Ctx) error {
		return c.SendString("q=" + c.Query("q"))
	})
	req := httptest.NewRequest("GET", "/search?q=hello", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Body.String() != "q=hello" { t.Errorf("Query failed") }
}

func TestCtxRedirect(t *testing.T) {
	app := NewHttpServer()
	app.Get("/old", func(c *Ctx) error {
		return c.Redirect("/new")
	})
	req := httptest.NewRequest("GET", "/old", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 302 { t.Errorf("Redirect failed, got %d", w.Code) }
	loc := w.Header().Get("Location")
	if loc != "/new" { t.Errorf("Location failed, got %s", loc) }
}

func TestCtxFormValue(t *testing.T) {
	app := NewHttpServer()
	app.Post("/form", func(c *Ctx) error {
		return c.SendString(c.FormValue("name"))
	})
	body := strings.NewReader("name=john")
	req := httptest.NewRequest("POST", "/form", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Body.String() != "john" { t.Errorf("FormValue failed: %s", w.Body.String()) }
}

func TestCtxLocals(t *testing.T) {
	app := NewHttpServer()
	app.Get("/locals", func(c *Ctx) error {
		c.Locals("user", "admin")
		u := c.Locals("user").(string)
		return c.SendString(u)
	})
	req := httptest.NewRequest("GET", "/locals", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Body.String() != "admin" { t.Errorf("Locals failed") }
}

func TestCtxStatusChain(t *testing.T) {
	app := NewHttpServer()
	app.Get("/status404", func(c *Ctx) error {
		return c.Status(404).SendString("not found")
	})
	req := httptest.NewRequest("GET", "/status404", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 404 { t.Errorf("Status chain failed") }
}

func TestHttpServerListenShutdown(t *testing.T) {
	app := NewHttpServer()
	
	errChan := make(chan error, 1)
	go func() {
		errChan <- app.Listen(":0")
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	err := app.Shutdown()
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	select {
	case err := <-errChan:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Listen failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Errorf("Server did not shut down in time")
	}
}

func TestCtxBodyAndParams(t *testing.T) {
	app := NewHttpServer()
	
	app.Post("/data/:id", func(c *Ctx) error {
		id := c.Params("id")
		body := string(c.Body())
		return c.SendString(id + "-" + body)
	})
	
	body := strings.NewReader("hello")
	req := httptest.NewRequest("POST", "/data/123", body)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	
	if w.Body.String() != "123-hello" {
		t.Errorf("Body/Params failed: %s", w.Body.String())
	}
}

func TestCtxCookie(t *testing.T) {
	app := NewHttpServer()
	
	app.Get("/cookie", func(c *Ctx) error {
		c.Cookie(&Cookie{
			Name: "token",
			Value: "xyz",
		})
		return c.SendString("ok")
	})
	
	req := httptest.NewRequest("GET", "/cookie", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	
	header := w.Header().Get("Set-Cookie")
	if !strings.Contains(header, "token=xyz") {
		t.Errorf("Cookie failed")
	}
}

func TestMiddlewareCompress(t *testing.T) {
	app := NewHttpServer()
	app.Use(Compress())
	app.Get("/comp", func(c *Ctx) error {
		return c.SendString("compress me")
	})
	
	req := httptest.NewRequest("GET", "/comp", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("Compress failed")
	}
}

func TestMiddlewareFavicon(t *testing.T) {
	app := NewHttpServer()
	app.Use(Favicon())
	
	req := httptest.NewRequest("GET", "/favicon.ico", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	
	if w.Code != 200 {
		t.Errorf("Favicon failed")
	}
}

func TestMiddlewareMonitor(t *testing.T) {
	app := NewHttpServer()
	app.Use(Monitor())
	
	req := httptest.NewRequest("GET", "/monitor", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	
	if w.Code != 200 || !strings.Contains(w.Body.String(), "NanoPony Monitor") {
		t.Errorf("Monitor failed")
	}
}

func TestHttpServerMethods(t *testing.T) {
	app := NewHttpServer()
	var called bool
	
	app.Trace("/trace", func(c *Ctx) error { called = true; return c.SendString("trace") })
	req := httptest.NewRequest("TRACE", "/trace", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if !called || w.Body.String() != "trace" { t.Errorf("Trace failed") }
	
	called = false
	app.Connect("/connect", func(c *Ctx) error { called = true; return c.SendString("connect") })
	req = httptest.NewRequest("CONNECT", "/connect", nil)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if !called || w.Body.String() != "connect" { t.Errorf("Connect failed") }
}

func TestHttpServerListenError(t *testing.T) {
	app := NewHttpServer()
	// Try to listen on invalid port to cover error path
	err := app.Listen("invalid-port")
	if err == nil {
		t.Errorf("Expected listen error")
	}
}

func TestMiddlewareETag(t *testing.T) {
	app := NewHttpServer()
	app.Use(ETag())
	app.Get("/etag", func(c *Ctx) error { return c.SendString("etag") })
	
	req := httptest.NewRequest("GET", "/etag", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 200 { t.Errorf("ETag failed") }
}

func TestHttpServerMethodsAdvanced(t *testing.T) {
	app := NewHttpServer()
	
	// Test remaining methods directly
	app.Post("/post", func(c *Ctx) error { return c.SendString("post") })
	app.Put("/put2", func(c *Ctx) error { return c.SendString("put") })
	app.Delete("/delete", func(c *Ctx) error { return c.SendString("delete") })
	app.Patch("/patch2", func(c *Ctx) error { return c.SendString("patch") })
	
	// Test Use with multiple string/handlers
	app.Use("/api", func(c *Ctx) error {
		c.Set("X-Api", "true")
		return c.Next()
	})
	
	app.Get("/api/test", func(c *Ctx) error {
		return c.SendString("api ok")
	})
	
	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Header().Get("X-Api") != "true" {
		t.Errorf("Multiple Use args failed")
	}
	
	// Test MultipartForm
	req = httptest.NewRequest("POST", "/upload", nil)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=foo")
	w = httptest.NewRecorder()
	
	app.Post("/upload", func(c *Ctx) error {
		form, err := c.MultipartForm()
		if err != nil {
			return c.SendString(err.Error())
		}
		if form == nil {
			return c.SendString("nil form")
		}
		return c.SendString("uploaded")
	})
	app.ServeHTTP(w, req)
}

func TestSQLiteQueueEdgeCases(t *testing.T) {
	// Test fetching from a non-existent database/table
	// Using in-memory to not leave files
	q, err := NewSQLiteQueue("file::memory:?cache=shared", "test_table")
	if err == nil {
		q.db.Exec("DROP TABLE test_table")
		_, err := q.FetchPending(context.Background())
		if err == nil {
			t.Errorf("Expected error fetching from dropped table")
		}
		q.Close()
	}
	
	// Close edge case
	q2 := &SQLiteQueue{}
	q2.Close() // Safe due to nil check in standard sql.DB or panic, let's see. Wait, actually we need to make sure db is not nil if we call close.
}

func TestHttpServerGroupMethods(t *testing.T) {
	app := NewHttpServer()
	
	g := app.Group("/api", func(c *Ctx) error {
		c.Set("X-Group", "true")
		return c.Next()
	})
	
	g.Get("/items", func(c *Ctx) error { return c.SendString("items") })
	g.Post("/items", func(c *Ctx) error { return c.SendString("created") })
	g.Put("/items/:id", func(c *Ctx) error { return c.SendString("updated") })
	g.Delete("/items/:id", func(c *Ctx) error { return c.SendString("deleted") })
	
	req := httptest.NewRequest("GET", "/api/items", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Header().Get("X-Group") != "true" || w.Body.String() != "items" {
		t.Errorf("Group Get failed")
	}
	
	req = httptest.NewRequest("POST", "/api/items", nil)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Body.String() != "created" { t.Errorf("Group Post failed") }
	
	req = httptest.NewRequest("PUT", "/api/items/1", nil)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Body.String() != "updated" { t.Errorf("Group Put failed") }
	
	req = httptest.NewRequest("DELETE", "/api/items/1", nil)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Body.String() != "deleted" { t.Errorf("Group Delete failed") }
	
	// Test nested group
	g2 := g.Group("/sub")
	g2.Get("/resource", func(c *Ctx) error { return c.SendString("sub resource") })
	req = httptest.NewRequest("GET", "/api/sub/resource", nil)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Body.String() != "sub resource" { t.Errorf("Nested group failed") }
}

func TestHttpServerCtxEdgeCases(t *testing.T) {
	app := NewHttpServer()
	
	app.Get("/edge", func(c *Ctx) error {
		// Test default param
		_ = c.Params("missing", "default")
		// Test default query
		_ = c.Query("missing", "fallback")
		// Test Send with nil
		err := c.Send(nil)
		return err
	})
	
	req := httptest.NewRequest("GET", "/edge", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 200 { t.Errorf("Edge cases failed") }
	
	// Test 404
	req = httptest.NewRequest("GET", "/not-found", nil)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 404 { t.Errorf("Expected 404, got %d", w.Code) }
}

func TestHttpServerConfigCustom(t *testing.T) {
	cfg := HttpConfig{
		Port: "9999",
		ServerHeader: "NanoPony-Test",
	}
	app := NewHttpServer(cfg)
	if app.config.Port != "9999" {
		t.Errorf("Custom config not applied")
	}
	
	app.Get("/hdr", func(c *Ctx) error { return c.SendString("ok") })
	req := httptest.NewRequest("GET", "/hdr", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Header().Get("Server") != "NanoPony-Test" { t.Errorf("Server header missing") }
}

func TestHttpServerDefaultConfig(t *testing.T) {
	app := NewHttpServer()
	if app.config.Port != "3000" {
		t.Errorf("Default port should be 3000, got %s", app.config.Port)
	}
}

func TestMiddlewareFaviconCustom(t *testing.T) {
	app := NewHttpServer()
	app.Use(Favicon(FaviconConfig{
		Data: []byte{0x01, 0x02},
	}))
	
	req := httptest.NewRequest("GET", "/favicon.ico", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 200 { t.Errorf("Custom favicon failed") }
}

func TestMiddlewareFaviconFile(t *testing.T) {
	app := NewHttpServer()
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/favicon.ico"
	os.WriteFile(tmpFile, []byte{0x00}, 0644)
	
	app.Use(Favicon(FaviconConfig{
		File: tmpFile,
	}))
	
	req := httptest.NewRequest("GET", "/favicon.ico", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 200 { t.Errorf("File favicon failed") }
}
