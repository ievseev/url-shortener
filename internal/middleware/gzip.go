package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"
)

var compressibleContentTypes = map[string]bool{
	"application/json": true,
	"text/html":        true,
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// обрабатываем сжатые входящие данные
		if r.Header.Get("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Failed to decompress request", http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = gz
		}

		// проверяем поддержку gzip клиентом
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// создаем обертку для условного сжатия
		wrapper := &responseWrapper{
			ResponseWriter: w,
		}

		// передаем управление следующему обработчику
		next.ServeHTTP(wrapper, r)

		// обязательно закрываем gzip writer после выполнения
		wrapper.Close()
	})
}

type responseWrapper struct {
	http.ResponseWriter
	gzWriter      *gzip.Writer
	headerWritten bool
}

func (w *responseWrapper) WriteHeader(statusCode int) {
	if w.headerWritten {
		return
	}

	contentType := w.Header().Get("Content-Type")
	shouldCompress := false

	// проверяем, является ли Content-Type сжимаемым
	for ct := range compressibleContentTypes {
		if strings.Contains(contentType, ct) {
			shouldCompress = true
			break
		}
	}

	// инициализируем gzip writer только если нужно сжимать
	if shouldCompress {
		var err error
		w.gzWriter, err = gzip.NewWriterLevel(w.ResponseWriter, gzip.BestSpeed)
		if err == nil {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Del("Content-Length")
		} else {
			w.gzWriter = nil // на случай ошибки
		}
	}

	w.ResponseWriter.WriteHeader(statusCode)
	w.headerWritten = true
}

func (w *responseWrapper) Write(b []byte) (int, error) {
	if !w.headerWritten {
		w.WriteHeader(http.StatusOK)
	}

	// записываем через gzip writer если он был создан
	if w.gzWriter != nil {
		return w.gzWriter.Write(b)
	}

	return w.ResponseWriter.Write(b)
}

// Close закрывает gzip writer и записывает оставшиеся данные
func (w *responseWrapper) Close() {
	if w.gzWriter != nil {
		w.gzWriter.Close()
	}
}
