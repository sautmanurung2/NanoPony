package nanopony

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
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
		return c.SendString("ok")
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
