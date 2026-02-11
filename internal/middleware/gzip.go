package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

const (
	gzipEncoding       = "gzip"
	contentEncodingKey = "Content-Encoding"
	acceptEncodingKey  = "Accept-Encoding"
	contentTypeKey     = "Content-Type"
	varyKey            = "Vary"
)

// compressibleContentTypes определяет типы контента, которые можно сжимать
var compressibleContentTypes = map[string]bool{
	"application/json": true,
	"text/html":        true,
}

// gzipWriter оборачивает http.ResponseWriter для записи сжатых данных
type gzipWriter struct {
	http.ResponseWriter
	writer   *gzip.Writer
	compress bool
}

// Write записывает сжатые данные если включено сжатие
func (g *gzipWriter) Write(data []byte) (int, error) {
	if !g.compress {
		return g.ResponseWriter.Write(data)
	}
	return g.writer.Write(data)
}

// WriteHeader перехватывает запись заголовков для настройки сжатия
func (g *gzipWriter) WriteHeader(statusCode int) {
	// Определяем, нужно ли включать сжатие на основе Content-Type
	contentType := g.ResponseWriter.Header().Get(contentTypeKey)
	if contentType != "" {
		// Извлекаем только MIME-тип без параметров
		parts := strings.Split(contentType, ";")
		mimeType := strings.TrimSpace(parts[0])

		// Включаем сжатие только для поддерживаемых типов контента
		if compressibleContentTypes[mimeType] {
			g.compress = true
			g.ResponseWriter.Header().Set(contentEncodingKey, gzipEncoding)
		}
	}

	g.ResponseWriter.WriteHeader(statusCode)
}

// Close закрывает gzip writer
func (g *gzipWriter) Close() error {
	if g.writer != nil {
		return g.writer.Close()
	}
	return nil
}

// gzipReader оборачивает io.Reader для чтения сжатых данных
type gzipReader struct {
	io.ReadCloser
	reader *gzip.Reader
}

// Read читает и распаковывает данные
func (g *gzipReader) Read(p []byte) (int, error) {
	return g.reader.Read(p)
}

// Close закрывает оба reader'а
func (g *gzipReader) Close() error {
	g.reader.Close()
	return g.ReadCloser.Close()
}

// supportsGzip проверяет, поддерживает ли клиент gzip
func supportsGzip(r *http.Request) bool {
	acceptEncoding := r.Header.Get(acceptEncodingKey)
	return strings.Contains(acceptEncoding, gzipEncoding)
}

// hasGzipEncoding проверяет, сжато ли тело запроса
func hasGzipEncoding(r *http.Request) bool {
	contentEncoding := r.Header.Get(contentEncodingKey)
	return contentEncoding == gzipEncoding
}

// Middleware обрабатывает gzip сжатие/распаковку
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Обработка входящих сжатых запросов
		if hasGzipEncoding(r) {
			reader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Failed to create gzip reader", http.StatusBadRequest)
				return
			}

			// Заменяем тело запроса на распакованное
			r.Body = &gzipReader{
				ReadCloser: r.Body,
				reader:     reader,
			}
		}

		// Обработка исходящих ответов - только если клиент поддерживает gzip
		if supportsGzip(r) {
			// Добавляем заголовок Vary
			w.Header().Set(varyKey, acceptEncodingKey)

			// Создаем gzip writer
			gzWriter, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
			if err != nil {
				http.Error(w, "Failed to create gzip writer", http.StatusInternalServerError)
				return
			}
			defer gzWriter.Close()

			// Оборачиваем ResponseWriter
			gw := &gzipWriter{
				ResponseWriter: w,
				writer:         gzWriter,
				compress:       false, // будет установлено в WriteHeader только для поддерживаемых типов
			}

			// Передаем управление следующему обработчику
			next.ServeHTTP(gw, r)
		} else {
			// Если клиент не поддерживает gzip, обрабатываем как обычно
			// В этом случае ответ НЕ будет сжат
			next.ServeHTTP(w, r)
		}
	})
}
