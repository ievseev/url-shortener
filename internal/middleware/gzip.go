package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// compressibleContentTypes определяет типы контента, которые могут быть сжаты
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

		// проверяем поддержку gzip клиентом
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			// если клиент не поддерживает gzip, передаем управление без сжатия
			next.ServeHTTP(w, r)
			return
		}

		// создаем обертку для перехвата и анализа Content-Type
		wrapper := &responseWrapper{
			ResponseWriter: w,
			request:        r,
		}

		next.ServeHTTP(wrapper, r)
	})
}

// responseWrapper обертка для анализа Content-Type и условного сжатия
type responseWrapper struct {
	http.ResponseWriter
	request        *http.Request
	gzWriter       *gzip.Writer
	headerWritten  bool
	shouldCompress bool
}

func (w *responseWrapper) WriteHeader(statusCode int) {
	if w.headerWritten {
		return
	}

	contentType := w.Header().Get("Content-Type")

	// проверяем, нужно ли сжимать: Content-Type должен быть в списке сжимаемых
	// И клиент должен поддерживать gzip (это уже проверено выше)
	for ct := range compressibleContentTypes {
		if strings.Contains(contentType, ct) {
			w.shouldCompress = true
			break
		}
	}

	// если нужно сжимать, инициализируем gzip writer
	if w.shouldCompress {
		var err error
		w.gzWriter, err = gzip.NewWriterLevel(w.ResponseWriter, gzip.BestSpeed)
		if err != nil {
			w.shouldCompress = false
		} else {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Del("Content-Length")
		}
	}

	w.ResponseWriter.WriteHeader(statusCode)
	w.headerWritten = true
}

func (w *responseWrapper) Write(b []byte) (int, error) {
	if !w.headerWritten {
		w.WriteHeader(http.StatusOK)
	}

	// если должны сжимать и gzip writer инициализирован
	if w.shouldCompress && w.gzWriter != nil {
		return w.gzWriter.Write(b)
	}

	// иначе записываем как обычно
	return w.ResponseWriter.Write(b)
}

// Close закрывает gzip writer, если он был создан
func (w *responseWrapper) Close() error {
	if w.gzWriter != nil {
		return w.gzWriter.Close()
	}
	return nil
}
