package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMimeTypes(t *testing.T) {
	testCases := map[string]string{
		"index.html":          "text/html; charset=utf-8",
		"javascript/admin.js": "application/javascript",
		"css/admin.css":       "text/css; charset=utf-8",
		"images/gopher.jpg":   "image/jpeg",
	}

	for path, expectedMime := range testCases {
		r, _ := http.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		serveFile(w, r, path)

		var mime string
		mimes := w.Header()["Content-Type"]
		if len(mimes) == 1 {
			mime = mimes[0]
		}

		if mime != expectedMime {
			t.Errorf("Expected mime type on '%s' to be '%s'; got '%s'", path, expectedMime, mime)
		}
	}
}
