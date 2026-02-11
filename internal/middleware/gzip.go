package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// compressibleContentTypes определяет типы контента, которые будут сжиматься
var compressibleContentTypes = map[string]bool{
	"application/json": true,
	"text/html":        true,
}

// gzipWriter обертка для http.ResponseWriter с поддержкой gzip-сжатия
type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	// записываем данные через gzip-writer
	return w.Writer.Write(b)
}

// GzipMiddleware создает middleware для обработки gzip-сжатия
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

		// проверяем поддержку gzip клиентом для ответа
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// создаем gzip.Writer для сжатия ответа
		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		defer gz.Close()

		// создаем обертку для перехвата заголовка Content-Type
		wrapper := &responseWrapper{
			ResponseWriter: w,
			gzWriter:       gz,
		}

		// передаем управление следующему обработчику
		next.ServeHTTP(wrapper, r)
	})
}

// responseWrapper обертка для перехвата установки заголовков
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

	// проверяем, нужно ли сжимать данный тип контента
	for ct := range compressibleContentTypes {
		if strings.Contains(contentType, ct) {
			shouldCompress = true
			break
		}
	}

	if shouldCompress {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length") // удаляем, так как длина изменится после сжатия
	}

	w.ResponseWriter.WriteHeader(statusCode)
	w.headerWritten = true
}

func (w *responseWrapper) Write(b []byte) (int, error) {
	if !w.headerWritten {
		w.WriteHeader(http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	shouldCompress := false

	// проверяем, нужно ли сжимать данный тип контента
	for ct := range compressibleContentTypes {
		if strings.Contains(contentType, ct) {
			shouldCompress = true
			break
		}
	}

	if shouldCompress && w.Header().Get("Content-Encoding") == "gzip" {
		return w.gzWriter.Write(b)
	}

	return w.ResponseWriter.Write(b)
}
