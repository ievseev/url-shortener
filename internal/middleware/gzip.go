package gzip

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

const (
	headerAcceptEncoding  = "Accept-Encoding"
	headerContentEncoding = "Content-Encoding"
	headerContentType     = "Content-Type"
	headerVary            = "Vary"

	encodingGzip = "gzip"
)

var pool = sync.Pool{
	New: func() any { return gzip.NewWriter(io.Discard) },
}

// Middleware adds gzip support for requests (Content-Encoding: gzip)
// and responses (Accept-Encoding: gzip) for application/json and text/html.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1) Decompress request if needed
		if hasToken(r.Header.Get(headerContentEncoding), encodingGzip) {
			zr, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			defer zr.Close()

			r.Body = &readCloser{
				Reader: zr,
				close: func() error {
					_ = zr.Close()
					return r.Body.Close()
				},
			}
			// request body is now decompressed; remove content-encoding
			r.Header.Del(headerContentEncoding)
		}

		// 2) Compress response if client supports it
		if !hasToken(r.Header.Get(headerAcceptEncoding), encodingGzip) {
			next.ServeHTTP(w, r)
			return
		}

		// Ensure caches vary by encoding support
		w.Header().Add(headerVary, headerAcceptEncoding)

		gzw := pool.Get().(*gzip.Writer)
		defer pool.Put(gzw)

		gzw.Reset(w)
		defer func() { _ = gzw.Close() }()

		cw := &compressResponseWriter{
			ResponseWriter: w,
			gzw:            gzw,
		}

		next.ServeHTTP(cw, r)
	})
}

type readCloser struct {
	io.Reader
	close func() error
}

func (rc *readCloser) Close() error { return rc.close() }

type compressResponseWriter struct {
	http.ResponseWriter
	gzw *gzip.Writer

	wroteHeader bool
	compress    bool
}

func (w *compressResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true

	// Decide based on Content-Type the handler set
	ct := w.Header().Get(headerContentType)
	mt := mimeType(ct)

	if mt == "application/json" || mt == "text/html" {
		w.compress = true
		w.Header().Set(headerContentEncoding, encodingGzip)
		// When encoding changes, Content-Length is no longer valid
		w.Header().Del("Content-Length")
	}

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *compressResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		// If handler didn't call WriteHeader explicitly, infer now
		w.WriteHeader(http.StatusOK)
	}

	if !w.compress {
		return w.ResponseWriter.Write(p)
	}
	return w.gzw.Write(p)
}

func hasToken(headerValue, token string) bool {
	// Very small token parser: accepts "gzip" in "gzip, deflate, br"
	for _, part := range strings.Split(headerValue, ",") {
		if strings.EqualFold(strings.TrimSpace(strings.SplitN(part, ";", 2)[0]), token) {
			return true
		}
	}
	return false
}

func mimeType(contentType string) string {
	// contentType can be: "application/json; charset=utf-8"
	if contentType == "" {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
}
