package nanopony

import (
	"net/http"
	"testing"
	"time"
)

func TestHttpServerListenShutdown_Race(t *testing.T) {
	app := NewHttpServer()
	
	// We want to trigger Shutdown between app.server = srv and srv.ListenAndServe()
	// This is hard to hit reliably, so we'll just try to see if it happens.
	
	errChan := make(chan error, 1)
	go func() {
		errChan <- app.Listen(":0")
	}()

	// Very short sleep to increase chance of hitting the race
	time.Sleep(1 * time.Millisecond)

	err := app.Shutdown()
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	select {
	case err := <-errChan:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Listen failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Errorf("Server did not shut down in time (likely hit the race condition)")
	}
}
