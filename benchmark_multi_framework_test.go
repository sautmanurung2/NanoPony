package nanopony

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/kataras/iris/v12"
	"github.com/labstack/echo/v4"
)

// ==================== MULTI-FRAMEWORK BENCHMARKS ====================

// BenchmarkFrameworks_Setup measures the overhead of setting up the frameworks.
func BenchmarkFrameworks_Setup(b *testing.B) {
	b.Run("NanoPony", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ResetConfig()
			config := NewConfig()
			fw := NewFramework().
				WithConfig(config).
				WithWorkerPool(10, 100).
				Build()
			if fw == nil {
				b.Fatal("failed to build NanoPony")
			}
		}
	})

	b.Run("Fiber", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			app := fiber.New(fiber.Config{
				DisableStartupMessage: true,
			})
			if app == nil {
				b.Fatal("failed to build Fiber")
			}
		}
	})

	b.Run("Echo", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			e := echo.New()
			e.HideBanner = true
			e.HidePort = true
			if e == nil {
				b.Fatal("failed to build Echo")
			}
		}
	})

	b.Run("Iris", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			app := iris.New()
			app.Logger().SetLevel("disable")
			if app == nil {
				b.Fatal("failed to build Iris")
			}
		}
	})
}

// BenchmarkFrameworks_Throughput measures processing speed for simple tasks.
func BenchmarkFrameworks_Throughput(b *testing.B) {
	ctx := context.Background()

	b.Run("NanoPony", func(b *testing.B) {
		ResetConfig()
		config := NewConfig()
		fw := NewFramework().
			WithConfig(config).
			WithWorkerPool(runtime.NumCPU(), b.N+1).
			Build()

		done := make(chan struct{}, b.N)
		fw.Start(ctx, func(ctx context.Context, job Job) error {
			done <- struct{}{}
			return nil
		})

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			fw.WorkerPool.Submit(ctx, Job{ID: fmt.Sprintf("%d", i)})
		}

		for i := 0; i < b.N; i++ {
			<-done
		}

		b.StopTimer()
		fw.Shutdown(ctx)
	})

	b.Run("Fiber", func(b *testing.B) {
		app := fiber.New(fiber.Config{
			DisableStartupMessage: true,
		})
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendStatus(200)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			resp, err := app.Test(req, -1)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
		b.StopTimer()
	})

	b.Run("Echo", func(b *testing.B) {
		e := echo.New()
		e.HideBanner = true
		e.HidePort = true
		e.GET("/test", func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				b.Fatalf("expected 200, got %d", rec.Code)
			}
		}
		b.StopTimer()
	})

	b.Run("Iris", func(b *testing.B) {
		app := iris.New()
		app.Logger().SetLevel("disable")
		app.Get("/test", func(ctx iris.Context) {
			ctx.StatusCode(iris.StatusOK)
		})

		if err := app.Build(); err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				b.Fatalf("expected 200, got %d", rec.Code)
			}
		}
		b.StopTimer()
	})
}

// TestFrameworks_MemoryUsage measures idle memory usage comparison.
func TestFrameworks_MemoryUsage(t *testing.T) {
	runMemoryTest := func(name string, initFn func()) {
		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)

		initFn()

		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)

		diff := int64(m2.Alloc) - int64(m1.Alloc)
		t.Logf("%s Idle Memory: %d KB (Alloc: %d KB)", name, diff/1024, m2.Alloc/1024)
	}

	runMemoryTest("NanoPony", func() {
		ResetConfig()
		config := NewConfig()
		fw := NewFramework().
			WithConfig(config).
			WithWorkerPool(10, 100).
			Build()
		fw.Start(context.Background(), func(ctx context.Context, job Job) error { return nil })
	})

	runMemoryTest("Fiber", func() {
		fiber.New(fiber.Config{DisableStartupMessage: true})
	})

	runMemoryTest("Echo", func() {
		e := echo.New()
		e.HideBanner = true
	})

	runMemoryTest("Iris", func() {
		app := iris.New()
		app.Logger().SetLevel("disable")
	})
}
