package shortenurlpost

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/ievseev/url-shortener/internal/handler/shortenurlpost/mocks"
	urlshortenerservice "github.com/ievseev/url-shortener/internal/service/urlshortener"
)

func TestShortenUrlPostHandler_Handle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	baseURL := "http://localhost:8080"

	tests := []struct {
		name               string
		requestBody        string
		contentType        string
		mockSetup          func(mockURLShortener *mocks.MockUrlShortener)
		expectedStatus     int
		expectedBodyPrefix string
		expectError        bool
	}{
		{
			name:        "успешное сокращение URL",
			requestBody: "https://example.com/very/long/url",
			contentType: "text/plain",
			mockSetup: func(mockURLShortener *mocks.MockUrlShortener) {
				mockURLShortener.EXPECT().
					Shorten(gomock.Any(), "https://example.com/very/long/url").
					Return("abc123", nil)
			},
			expectedStatus:     http.StatusCreated,
			expectedBodyPrefix: "http://localhost:8080/abc123",
		},
		{
			name:        "неправильный Content-Type",
			requestBody: "https://example.com",
			contentType: "application/json",
			mockSetup: func(mockURLShortener *mocks.MockUrlShortener) {
				// мок не должен вызываться
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "отсутствует Content-Type",
			requestBody: "https://example.com",
			contentType: "",
			mockSetup: func(mockURLShortener *mocks.MockUrlShortener) {
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "ошибка при сокращении URL",
			requestBody: "https://example.com",
			contentType: "text/plain",
			mockSetup: func(mockURLShortener *mocks.MockUrlShortener) {
				mockURLShortener.EXPECT().
					Shorten(gomock.Any(), "https://example.com").
					Return("", errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:        "конфликт при повторном сокращении URL",
			requestBody: "https://example.com",
			contentType: "text/plain",
			mockSetup: func(mockURLShortener *mocks.MockUrlShortener) {
				mockURLShortener.EXPECT().
					Shorten(gomock.Any(), "https://example.com").
					Return("abc123", urlshortenerservice.ErrorURLConflict)
			},
			expectedStatus:     http.StatusConflict,
			expectedBodyPrefix: "http://localhost:8080/abc123",
		},
		{
			name:        "пустое тело запроса",
			requestBody: "",
			contentType: "text/plain",
			mockSetup: func(mockURLShortener *mocks.MockUrlShortener) {
				mockURLShortener.EXPECT().
					Shorten(gomock.Any(), "").
					Return("empty123", nil)
			},
			expectedStatus:     http.StatusCreated,
			expectedBodyPrefix: "http://localhost:8080/empty123",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Создаем мок
			mockURLShortener := mocks.NewMockUrlShortener(ctrl)
			tc.mockSetup(mockURLShortener)

			// Создаем хендлер с моком
			handler := New(baseURL, mockURLShortener, slog.Default())

			// Создаем HTTP запрос
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.requestBody))
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}

			// Создаем ResponseRecorder для записи ответа
			rr := httptest.NewRecorder()

			// Вызываем хендлер
			handler.Handle(rr, req)

			// Проверяем статус код
			if rr.Code != tc.expectedStatus {
				t.Errorf("ожидался статус %d, получен %d", tc.expectedStatus, rr.Code)
			}

			// Проверяем Content-Type для успешных случаев
			if tc.expectedStatus == http.StatusCreated || tc.expectedStatus == http.StatusConflict {
				contentType := rr.Header().Get("Content-Type")
				if contentType != "text/plain" {
					t.Errorf("ожидался Content-Type 'text/plain', получен '%s'", contentType)
				}

				// Проверяем тело ответа
				body := rr.Body.String()
				if tc.expectedBodyPrefix != "" && body != tc.expectedBodyPrefix {
					t.Errorf("ожидался ответ '%s', получен '%s'", tc.expectedBodyPrefix, body)
				}
			}
		})
	}
}
