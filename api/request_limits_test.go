package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mockdb "github.com/faelic/monierave/db/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestOrdinaryRequestBodyLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	server := newTestServer(t, mockdb.NewMockStore(ctrl))
	body, err := json.Marshal(map[string]string{
		"username":  "favour",
		"password":  "safe-password",
		"full_name": strings.Repeat("a", maxJSONBodySize),
		"email":     "favour@example.com",
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"code":"request_body_too_large"`)
}

func TestJSONRequestsRejectUnknownFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	server := newTestServer(t, mockdb.NewMockStore(ctrl))
	body := []byte(`{
		"username":"favour",
		"password":"safe-password",
		"full_name":"Favour",
		"email":"favour@example.com",
		"is_admin":true
	}`)

	request := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}
