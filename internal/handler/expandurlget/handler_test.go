package expandurlget

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/mock/gomock"

	"github.com/ievseev/url-shortener/internal/handler/expandurlget/mocks"
)

func TestExpandUrlGetHandler_Handle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name           string
		shortURL       string
		mockSetup      func(mockURLShortener *mocks.MockUrlShortener)
		expectedStatus int
		expectedHeader string
	}{
		{
			name:     "успешное расширение URL",
			shortURL: "abc123",
			mockSetup: func(mockURLShortener *mocks.MockUrlShortener) {
				mockURLShortener.EXPECT().
					Expand(gomock.Any(), "abc123").
					Return("https://example.com", nil)
			},
			expectedStatus: http.StatusTemporaryRedirect,
			expectedHeader: "https://example.com",
		},
		{
			name:     "ошибка при расширении URL",
			shortURL: "invalid",
			mockSetup: func(mockURLShortener *mocks.MockUrlShortener) {
				mockURLShortener.EXPECT().
					Expand(gomock.Any(), "invalid").
					Return("", errors.New("URL not found"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedHeader: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Создаем мок
			mockURLShortener := mocks.NewMockUrlShortener(ctrl)
			tc.mockSetup(mockURLShortener)

			// Создаем хендлер с моком
			handler := New(mockURLShortener, slog.Default())

			// Создаем HTTP запрос
			req := httptest.NewRequest(http.MethodGet, "/expand/"+tc.shortURL, nil)

			// Настраиваем chi context для параметра URL
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.shortURL)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Создаем ResponseRecorder для записи ответа
			rr := httptest.NewRecorder()

			// Вызываем хендлер
			handler.Handle(rr, req)

			// Проверяем статус код
			if rr.Code != tc.expectedStatus {
				t.Errorf("ожидался статус %d, получен %d", tc.expectedStatus, rr.Code)
			}

			// Проверяем заголовок Location для успешных случаев
			if tc.expectedStatus == http.StatusTemporaryRedirect {
				location := rr.Header().Get("Location")
				if location != tc.expectedHeader {
					t.Errorf("ожидался заголовок Location '%s', получен '%s'", tc.expectedHeader, location)
				}
			}
		})
	}
}
