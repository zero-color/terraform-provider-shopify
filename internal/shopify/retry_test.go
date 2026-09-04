package shopify

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	goshopify "github.com/bold-commerce/go-shopify/v4"
)

func internalServerError(requestID string) goshopify.ResponseError {
	return goshopify.ResponseError{
		Status: 200,
		Errors: []string{
			shopifyInternalErrorMessage + " Request ID: " + requestID,
		},
	}
}

func TestRetryGraphQLRead_SucceedsAfterTransientErrors(t *testing.T) {
	t.Parallel()

	calls := 0
	err := retryGraphQLRead(context.Background(), zeroDelays(4), func() error {
		calls++
		if calls < 3 {
			return internalServerError("abc")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestRetryGraphQLRead_PermanentErrorReturnsImmediately(t *testing.T) {
	t.Parallel()

	permanentErr := errors.New("permanent failure")
	calls := 0

	err := retryGraphQLRead(context.Background(), zeroDelays(4), func() error {
		calls++
		return permanentErr
	})

	if !errors.Is(err, permanentErr) {
		t.Fatalf("expected permanent error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestRetryGraphQLRead_ReturnsLastErrorWhenRetriesExhausted(t *testing.T) {
	t.Parallel()

	lastErr := internalServerError("final")
	calls := 0

	err := retryGraphQLRead(context.Background(), zeroDelays(4), func() error {
		calls++
		return lastErr
	})

	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != lastErr.Error() {
		t.Fatalf("expected last error %q, got %q", lastErr.Error(), err.Error())
	}
	if calls != 5 {
		t.Fatalf("expected 5 calls, got %d", calls)
	}
}

func TestRetryGraphQLRead_ContextCanceledDuringDelay(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	requestID := "abc-request-id"

	err := retryGraphQLRead(ctx, []time.Duration{time.Hour}, func() error {
		calls++
		if calls == 1 {
			cancel()
		}
		return internalServerError(requestID)
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if !strings.Contains(err.Error(), requestID) {
		t.Fatalf("expected request ID %q in error, got %v", requestID, err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestRetryGraphQLRead_ContextCanceledBeforeAttempt(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0
	err := retryGraphQLRead(ctx, zeroDelays(4), func() error {
		calls++
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if calls != 0 {
		t.Fatalf("expected 0 calls, got %d", calls)
	}
}

func TestRetryGraphQLRead_MixedErrorsDoNotRetry(t *testing.T) {
	t.Parallel()

	mixedErr := goshopify.ResponseError{
		Status: 200,
		Errors: []string{
			"Internal error. Looks like something went wrong on our end.",
			"Some other error",
		},
	}
	calls := 0

	err := retryGraphQLRead(context.Background(), zeroDelays(4), func() error {
		calls++
		return mixedErr
	})

	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != mixedErr.Error() {
		t.Fatalf("expected mixed error %q, got %q", mixedErr.Error(), err.Error())
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestRetryGraphQLRead_Non200InternalErrorDoesNotRetry(t *testing.T) {
	t.Parallel()

	non200Err := goshopify.ResponseError{
		Status: 500,
		Errors: []string{
			"Internal error. Looks like something went wrong on our end.",
		},
	}
	calls := 0

	err := retryGraphQLRead(context.Background(), zeroDelays(4), func() error {
		calls++
		return non200Err
	})

	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != non200Err.Error() {
		t.Fatalf("expected non-200 error %q, got %q", non200Err.Error(), err.Error())
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestIsTransientGraphQLInternalError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "transient internal error",
			err:  internalServerError("abc"),
			want: true,
		},
		{
			name: "permanent error",
			err:  errors.New("permanent"),
			want: false,
		},
		{
			name: "mixed errors",
			err: goshopify.ResponseError{
				Status: 200,
				Errors: []string{
					"Internal error. Looks like something went wrong on our end.",
					"other",
				},
			},
			want: false,
		},
		{
			name: "non-200 status",
			err: goshopify.ResponseError{
				Status: 500,
				Errors: []string{
					"Internal error. Looks like something went wrong on our end.",
				},
			},
			want: false,
		},
		{
			name: "empty errors",
			err: goshopify.ResponseError{
				Status: 200,
				Errors: nil,
			},
			want: false,
		},
		{
			name: "pointer error",
			err: func() error {
				e := internalServerError("abc")
				return &e
			}(),
			want: true,
		},
		{
			name: "wrapped value error",
			err:  fmt.Errorf("query failed: %w", internalServerError("abc")),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isTransientGraphQLInternalError(tt.err); got != tt.want {
				t.Fatalf("isTransientGraphQLInternalError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func zeroDelays(n int) []time.Duration {
	delays := make([]time.Duration, n)
	return delays
}
