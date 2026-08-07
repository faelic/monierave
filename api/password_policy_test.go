package api

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type passwordBreachCheckerFunc func(context.Context, string) (bool, error)

func (function passwordBreachCheckerFunc) IsCompromised(
	ctx context.Context,
	password string,
) (bool, error) {
	return function(ctx, password)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func hibpResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestValidateNewPasswordBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "seven characters", password: "1234567", wantErr: true},
		{name: "eight characters", password: "12345678"},
		{name: "72 bytes", password: strings.Repeat("a", 72)},
		{name: "73 bytes", password: strings.Repeat("a", 73), wantErr: true},
		{name: "eight runes over 72 bytes", password: strings.Repeat("界", 25), wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateNewPassword(test.password)
			if test.wantErr {
				require.ErrorIs(t, err, ErrInvalidPasswordLength)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestHIBPPasswordBreachChecker(t *testing.T) {
	password := "correct horse battery staple"
	digest := fmt.Sprintf("%X", sha1.Sum([]byte(password)))
	prefix, suffix := digest[:5], digest[5:]
	var calls atomic.Int32

	checker := NewHIBPPasswordBreachChecker(
		"https://api.pwnedpasswords.test/range",
		time.Second,
		time.Hour,
	)
	checker.client.Transport = roundTripFunc(func(
		request *http.Request,
	) (*http.Response, error) {
		calls.Add(1)
		require.Equal(t, "/range/"+prefix, request.URL.Path)
		require.Equal(t, "true", request.Header.Get("Add-Padding"))
		return hibpResponse(fmt.Sprintf(
			"%s:42\n%s:0\n",
			suffix,
			strings.Repeat("A", 35),
		)), nil
	})

	compromised, err := checker.IsCompromised(context.Background(), password)
	require.NoError(t, err)
	require.True(t, compromised)

	compromised, err = checker.IsCompromised(context.Background(), password)
	require.NoError(t, err)
	require.True(t, compromised)
	require.Equal(t, int32(1), calls.Load(), "the second lookup should use the cache")
}

func TestHIBPPaddingAndMalformedResponses(t *testing.T) {
	password := "another safe password"
	digest := fmt.Sprintf("%X", sha1.Sum([]byte(password)))
	suffix := digest[5:]

	t.Run("zero count padding is clean", func(t *testing.T) {
		checker := NewHIBPPasswordBreachChecker(
			"https://api.pwnedpasswords.test/range",
			time.Second,
			time.Hour,
		)
		checker.client.Transport = roundTripFunc(func(
			_ *http.Request,
		) (*http.Response, error) {
			return hibpResponse(fmt.Sprintf("%s:0\n", suffix)), nil
		})
		compromised, err := checker.IsCompromised(context.Background(), password)
		require.NoError(t, err)
		require.False(t, compromised)
	})

	t.Run("malformed response fails closed", func(t *testing.T) {
		checker := NewHIBPPasswordBreachChecker(
			"https://api.pwnedpasswords.test/range",
			time.Second,
			time.Hour,
		)
		checker.client.Transport = roundTripFunc(func(
			_ *http.Request,
		) (*http.Response, error) {
			return hibpResponse("malformed"), nil
		})
		_, err := checker.IsCompromised(context.Background(), password)
		require.Error(t, err)
	})
}

func TestHIBPTimeout(t *testing.T) {
	checker := NewHIBPPasswordBreachChecker(
		"https://api.pwnedpasswords.test/range",
		5*time.Millisecond,
		time.Hour,
	)
	checker.client.Transport = roundTripFunc(func(
		request *http.Request,
	) (*http.Response, error) {
		<-request.Context().Done()
		return nil, request.Context().Err()
	})
	_, err := checker.IsCompromised(context.Background(), "timeout password")
	require.Error(t, err)
}

func TestAcceptNewPasswordMapsBreachCheckResults(t *testing.T) {
	tests := []struct {
		name        string
		compromised bool
		checkErr    error
		status      int
		code        string
	}{
		{
			name:        "compromised",
			compromised: true,
			status:      http.StatusBadRequest,
			code:        "password_compromised",
		},
		{
			name:     "unavailable",
			checkErr: errors.New("HIBP unavailable"),
			status:   http.StatusServiceUnavailable,
			code:     "password_breach_check_unavailable",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := &Server{
				passwordBreachChecker: passwordBreachCheckerFunc(func(
					context.Context,
					string,
				) (bool, error) {
					return test.compromised, test.checkErr
				}),
			}
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/users", nil)

			require.False(t, server.acceptNewPassword(ctx, "safe-password"))
			require.Equal(t, test.status, recorder.Code)
			require.Contains(t, recorder.Body.String(), `"code":"`+test.code+`"`)
		})
	}
}
