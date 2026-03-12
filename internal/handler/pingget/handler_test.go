package pingget

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerHandle(t *testing.T) {
	tests := []struct {
		name         string
		pingErr      error
		expectedCode int
	}{
		{
			name:         "success",
			expectedCode: http.StatusOK,
		},
		{
			name:         "failure",
			pingErr:      errors.New("ping failed"),
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := New(&stubPinger{err: tc.pingErr}, slog.Default())
			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			recorder := httptest.NewRecorder()

			handler.Handle(recorder, req)

			if recorder.Code != tc.expectedCode {
				t.Fatalf("expected status %d, got %d", tc.expectedCode, recorder.Code)
			}
		})
	}
}

type stubPinger struct {
	err error
}

func (s *stubPinger) Ping(ctx context.Context) error {
	return s.err
}
