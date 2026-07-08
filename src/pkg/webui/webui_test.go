package webui

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

// pathRecorder is an http.Handler that records the request path it was asked to
// serve, standing in for the real file server.
type pathRecorder struct{ served string }

func (p *pathRecorder) ServeHTTP(_ http.ResponseWriter, r *http.Request) {
	p.served = r.URL.Path
}

func newTestSPAHandler(rec *pathRecorder) *SPAHandler {
	mapFS := fstest.MapFS{
		"index.html":     {Data: []byte("<html>index</html>")},
		"assets/app.js":  {Data: []byte("console.log(1)")},
		"assets/app.css": {Data: []byte("body{}")},
	}
	return &SPAHandler{
		FileServer: rec,
		FileSystem: http.FS(mapFS),
	}
}

func TestSPAHandlerServesExistingFile(t *testing.T) {
	rec := &pathRecorder{}
	h := newTestSPAHandler(rec)

	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)

	if rec.served != "/assets/app.js" {
		t.Fatalf("expected file server to receive /assets/app.js, got %q", rec.served)
	}
}

func TestSPAHandlerFallsBackToIndexForUnknownRoute(t *testing.T) {
	rec := &pathRecorder{}
	h := newTestSPAHandler(rec)

	req := httptest.NewRequest(http.MethodGet, "/some/client/route", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)

	if rec.served != "/" {
		t.Fatalf("expected SPA fallback to rewrite path to /, got %q", rec.served)
	}
}
