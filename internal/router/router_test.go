package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gofiber/fiber/v2"
)

func newEmbeddedFrontendTestApp(t *testing.T) *fiber.App {
	t.Helper()

	uiFS := http.FS(fstest.MapFS{
		"index.html": {
			Data: []byte(`<!doctype html><script type="module" src="/admin/_app/immutable/entry/start.test.js"></script>`),
		},
		"setup/index.html": {
			Data: []byte(`<!doctype html><title>Setup</title>`),
		},
		"_app/immutable/entry/start.test.js": {
			Data: []byte(`console.log("ok");`),
		},
		"_app/immutable/assets/app.test.css": {
			Data: []byte(`body { color: black; }`),
		},
		"_app/immutable/chunks/module.test.wasm": {
			Data: []byte{0x00, 0x61, 0x73, 0x6d},
		},
	})

	app := fiber.New()
	registerEmbeddedFrontend(app, uiFS)
	return app
}

func TestEmbeddedFrontendServesExistingModuleWithJavaScriptMIME(t *testing.T) {
	app := newEmbeddedFrontendTestApp(t)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/admin/_app/immutable/entry/start.test.js", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/javascript; charset=utf-8" {
		t.Fatalf("expected JavaScript MIME, got %q", got)
	}
}

func TestEmbeddedFrontendServesWASMWithWASMMIME(t *testing.T) {
	app := newEmbeddedFrontendTestApp(t)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/admin/_app/immutable/chunks/module.test.wasm", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/wasm" {
		t.Fatalf("expected WASM MIME, got %q", got)
	}
}

func TestEmbeddedFrontendDoesNotFallbackToHTMLForMissingStaticAsset(t *testing.T) {
	app := newEmbeddedFrontendTestApp(t)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/admin/_app/immutable/entry/missing.js", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing static asset status 404, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got == "text/html; charset=utf-8" {
		t.Fatalf("missing static asset must not be served as HTML")
	}
}

func TestEmbeddedFrontendFallsBackForSPARoute(t *testing.T) {
	app := newEmbeddedFrontendTestApp(t)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/admin/docs/dev", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("expected HTML MIME for SPA fallback, got %q", got)
	}
}
