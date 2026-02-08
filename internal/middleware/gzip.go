package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"
)

// GzipDecompressMiddleware обрабатывает входящие сжатые запросы
func GzipDecompressMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Проверяем заголовок Content-Encoding
			contentEncoding := r.Header.Get("Content-Encoding")

			// Если запрос не сжат, передаем управление дальше
			if !strings.Contains(contentEncoding, "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			// Создаем gzip reader для декомпрессии тела запроса
			gzipReader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Failed to create gzip reader", http.StatusBadRequest)
				return
			}
			defer gzipReader.Close()

			// Заменяем Body запроса на декомпрессованное содержимое
			r.Body = gzipReader

			// Убираем заголовок Content-Encoding, так как теперь тело не сжато
			r.Header.Del("Content-Encoding")

			// Передаем управление следующему обработчику
			next.ServeHTTP(w, r)
		})
	}
}

// gzipResponseWriter - wrapper для ResponseWriter, который сжимает исходящие данные
type gzipResponseWriter struct {
	http.ResponseWriter
	gzipWriter *gzip.Writer
}

func newGzipResponseWriter(w http.ResponseWriter) *gzipResponseWriter {
	gzipWriter := gzip.NewWriter(w)
	return &gzipResponseWriter{
		ResponseWriter: w,
		gzipWriter:     gzipWriter,
	}
}

func (grw *gzipResponseWriter) Write(data []byte) (int, error) {
	// Записываем данные через gzip writer
	return grw.gzipWriter.Write(data)
}

func (grw *gzipResponseWriter) WriteHeader(statusCode int) {
	// Устанавливаем заголовки сжатия перед записью статуса
	grw.Header().Set("Content-Encoding", "gzip")
	grw.Header().Set("Vary", "Accept-Encoding")
	// Удаляем Content-Length, так как размер изменится после сжатия
	grw.Header().Del("Content-Length")

	grw.ResponseWriter.WriteHeader(statusCode)
}

func (grw *gzipResponseWriter) Close() error {
	return grw.gzipWriter.Close()
}

// GzipCompressMiddleware обрабатывает исходящие ответы для сжатия
func GzipCompressMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Проверяем, поддерживает ли клиент gzip
			acceptEncoding := r.Header.Get("Accept-Encoding")
			if !strings.Contains(acceptEncoding, "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			// Создаем wrapper для ResponseWriter
			gzipWriter := newGzipResponseWriter(w)
			defer gzipWriter.Close()

			// Создаем wrapper для определения Content-Type
			responseTypeWriter := &responseTypeDetector{
				gzipResponseWriter: gzipWriter,
				originalWriter:     w,
				shouldCompress:     false,
			}

			// Передаем управление следующему обработчику
			next.ServeHTTP(responseTypeWriter, r)
		})
	}
}

// responseTypeDetector определяет, нужно ли сжимать ответ на основе Content-Type
type responseTypeDetector struct {
	*gzipResponseWriter
	originalWriter http.ResponseWriter
	shouldCompress bool
	headerWritten  bool
}

func (rtd *responseTypeDetector) WriteHeader(statusCode int) {
	if rtd.headerWritten {
		return
	}
	rtd.headerWritten = true

	contentType := rtd.Header().Get("Content-Type")

	// Проверяем, нужно ли сжимать этот тип контента
	rtd.shouldCompress = rtd.shouldCompressContentType(contentType)

	if rtd.shouldCompress {
		rtd.gzipResponseWriter.WriteHeader(statusCode)
	} else {
		rtd.originalWriter.WriteHeader(statusCode)
	}
}

func (rtd *responseTypeDetector) Write(data []byte) (int, error) {
	if !rtd.headerWritten {
		rtd.WriteHeader(http.StatusOK)
	}

	if rtd.shouldCompress {
		return rtd.gzipResponseWriter.Write(data)
	}
	return rtd.originalWriter.Write(data)
}

func (rtd *responseTypeDetector) shouldCompressContentType(contentType string) bool {
	compressibleTypes := []string{
		"application/json",
		"text/html",
	}

	for _, compressible := range compressibleTypes {
		if strings.Contains(contentType, compressible) {
			return true
		}
	}
	return false
}
