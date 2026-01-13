package shorten_url_post

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/ievseev/url-shortener/internal/handler/shorten_url_post/mocks"
)

func TestShortenUrlPostHandler_Handle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name           string
		contentType    string
		body           string
		mockSetup      func(mockUrlShortener *mocks.MockUrlShortener)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:        "успешное сокращение URL",
			contentType: "text/plain",
			body:        "https://example.com",
			mockSetup: func(mockUrlShortener *mocks.MockUrlShortener) {
				mockUrlShortener.EXPECT().
					Shorten(gomock.Any(), "https://example.com").
					Return("abc123", nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   "http://example.com/abc123",
		},
		{
			name:        "ошибка при сокращении URL",
			contentType: "text/plain",
			body:        "invalid-url",
			mockSetup: func(mockUrlShortener *mocks.MockUrlShortener) {
				mockUrlShortener.EXPECT().
					Shorten(gomock.Any(), "invalid-url").
					Return("", errors.New("invalid URL"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "",
		},
		{
			name:           "неверный Content-Type",
			contentType:    "application/json",
			body:           "https://example.com",
			mockSetup:      func(mockUrlShortener *mocks.MockUrlShortener) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "",
		},
		{
			name:           "отсутствующий Content-Type",
			contentType:    "",
			body:           "https://example.com",
			mockSetup:      func(mockUrlShortener *mocks.MockUrlShortener) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "",
		},
		{
			name:        "Content-Type с дополнительными параметрами",
			contentType: "text/plain; charset=utf-8",
			body:        "https://example.com",
			mockSetup: func(mockUrlShortener *mocks.MockUrlShortener) {
				mockUrlShortener.EXPECT().
					Shorten(gomock.Any(), "https://example.com").
					Return("def456", nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   "http://example.com/def456",
		},
		{
			name:        "пустое тело запроса",
			contentType: "text/plain",
			body:        "",
			mockSetup: func(mockUrlShortener *mocks.MockUrlShortener) {
				mockUrlShortener.EXPECT().
					Shorten(gomock.Any(), "").
					Return("", errors.New("empty URL"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Создаем мок
			mockUrlShortener := mocks.NewMockUrlShortener(ctrl)
			tc.mockSetup(mockUrlShortener)

			// Создаем хендлер с моком
			handler := New(mockUrlShortener)

			// Создаем HTTP запрос
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			req.Host = "example.com"

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

			// Проверяем тело ответа для успешных случаев
			if tc.expectedStatus == http.StatusCreated {
				body := strings.TrimSpace(rr.Body.String())
				if body != tc.expectedBody {
					t.Errorf("ожидалось тело ответа '%s', получено '%s'", tc.expectedBody, body)
				}

				// Проверяем заголовок Content-Type
				contentType := rr.Header().Get("Content-Type")
				if contentType != "text/plain" {
					t.Errorf("ожидался Content-Type 'text/plain', получен '%s'", contentType)
				}
			}
		})
	}
}
