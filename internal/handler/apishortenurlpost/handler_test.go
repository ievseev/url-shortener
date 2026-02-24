package apishortenurlpost

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ievseev/url-shortener/internal/handler/apishortenurlpost/mocks"
)

func TestAPIShortenurlPostHandler_Handle_SuccessCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	baseURL := "http://localhost:8080"

	tests := []struct {
		name           string
		requestBody    *Request
		mockSetup      func(MockURLShortener *mocks.MockURLShortener)
		expectedResult *Response
	}{
		{
			name: "успешное сокращение URL",
			requestBody: &Request{
				URL: "https://example.com/very/long/url",
			},
			mockSetup: func(MockURLShortener *mocks.MockURLShortener) {
				MockURLShortener.EXPECT().
					Shorten(gomock.Any(), "https://example.com/very/long/url").
					Return("abc123", nil)
			},
			expectedResult: &Response{
				Result: "http://localhost:8080/abc123",
			},
		},
		{
			name: "пустой URL в запросе",
			requestBody: &Request{
				URL: "",
			},
			mockSetup: func(MockURLShortener *mocks.MockURLShortener) {
				MockURLShortener.EXPECT().
					Shorten(gomock.Any(), "").
					Return("empty123", nil)
			},
			expectedResult: &Response{
				Result: "http://localhost:8080/empty123",
			},
		},
		{
			name: "длинный URL с параметрами",
			requestBody: &Request{
				URL: "https://example.com/path?param1=value1&param2=value2",
			},
			mockSetup: func(MockURLShortener *mocks.MockURLShortener) {
				MockURLShortener.EXPECT().
					Shorten(gomock.Any(), "https://example.com/path?param1=value1&param2=value2").
					Return("param123", nil)
			},
			expectedResult: &Response{
				Result: "http://localhost:8080/param123",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			MockURLShortener := mocks.NewMockURLShortener(ctrl)
			tc.mockSetup(MockURLShortener)

			handler := New(baseURL, MockURLShortener, slog.Default())

			requestBody, err := json.Marshal(tc.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.Handle(rr, req)

			assert.Equal(t, http.StatusCreated, rr.Code)
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

			var actualResponse Response
			err = json.Unmarshal(rr.Body.Bytes(), &actualResponse)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedResult.Result, actualResponse.Result)
		})
	}
}

func TestAPIShortenurlPostHandler_Handle_ContentTypeValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	baseURL := "http://localhost:8080"

	tests := []struct {
		name        string
		requestBody *Request
		contentType string
	}{
		{
			name: "неправильный Content-Type text/plain",
			requestBody: &Request{
				URL: "https://example.com",
			},
			contentType: "text/plain",
		},
		{
			name: "неправильный Content-Type application/xml",
			requestBody: &Request{
				URL: "https://example.com",
			},
			contentType: "application/xml",
		},
		{
			name: "отсутствует Content-Type",
			requestBody: &Request{
				URL: "https://example.com",
			},
			contentType: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			MockURLShortener := mocks.NewMockURLShortener(ctrl)
			// мок не должен вызываться для неправильного Content-Type

			handler := New(baseURL, MockURLShortener, slog.Default())

			requestBody, err := json.Marshal(tc.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(requestBody))
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}

			rr := httptest.NewRecorder()
			handler.Handle(rr, req)

			assert.Equal(t, http.StatusBadRequest, rr.Code)
		})
	}
}

func TestAPIShortenurlPostHandler_Handle_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	baseURL := "http://localhost:8080"

	tests := []struct {
		name        string
		requestJSON string
	}{
		{
			name:        "невалидный JSON - незакрытая скобка",
			requestJSON: `{"url":"https://example.com"`,
		},
		{
			name:        "невалидный JSON - неправильные кавычки",
			requestJSON: `{'url':'https://example.com'}`,
		},
		{
			name:        "невалидный JSON - лишняя запятая",
			requestJSON: `{"url":"https://example.com",}`,
		},
		{
			name:        "пустая строка",
			requestJSON: ``,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			MockURLShortener := mocks.NewMockURLShortener(ctrl)
			// мок не должен вызываться для невалидного JSON

			handler := New(baseURL, MockURLShortener, slog.Default())

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(tc.requestJSON))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.Handle(rr, req)

			assert.Equal(t, http.StatusBadRequest, rr.Code)
		})
	}
}

func TestAPIShortenurlPostHandler_Handle_EmptyJSONObject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	baseURL := "http://localhost:8080"
	MockURLShortener := mocks.NewMockURLShortener(ctrl)

	MockURLShortener.EXPECT().
		Shorten(gomock.Any(), "").
		Return("empty123", nil)

	handler := New(baseURL, MockURLShortener, slog.Default())

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Handle(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	expectedResult := &Response{
		Result: "http://localhost:8080/empty123",
	}

	var actualResponse Response
	err := json.Unmarshal(rr.Body.Bytes(), &actualResponse)
	require.NoError(t, err)

	assert.Equal(t, expectedResult.Result, actualResponse.Result)
}

func TestAPIShortenurlPostHandler_Handle_ServiceErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	baseURL := "http://localhost:8080"

	tests := []struct {
		name         string
		requestBody  *Request
		serviceError error
	}{
		{
			name: "ошибка базы данных",
			requestBody: &Request{
				URL: "https://example.com",
			},
			serviceError: errors.New("database connection error"),
		},
		{
			name: "ошибка валидации URL в сервисе",
			requestBody: &Request{
				URL: "invalid-url",
			},
			serviceError: errors.New("invalid URL format"),
		},
		{
			name: "внутренняя ошибка сервиса",
			requestBody: &Request{
				URL: "https://example.com/test",
			},
			serviceError: errors.New("internal service error"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			MockURLShortener := mocks.NewMockURLShortener(ctrl)
			MockURLShortener.EXPECT().
				Shorten(gomock.Any(), tc.requestBody.URL).
				Return("", tc.serviceError)

			handler := New(baseURL, MockURLShortener, slog.Default())

			requestBody, err := json.Marshal(tc.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.Handle(rr, req)

			assert.Equal(t, http.StatusInternalServerError, rr.Code)
		})
	}
}
