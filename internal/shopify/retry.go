package shopify

import (
	"context"
	"errors"
	"strings"
	"time"

	goshopify "github.com/bold-commerce/go-shopify/v4"
)

const shopifyInternalErrorMessage = "Internal error. Looks like something went wrong on our end."

var defaultGraphQLReadRetryDelays = []time.Duration{
	time.Second,
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
}

func isTransientGraphQLInternalError(err error) bool {
	respErr, ok := responseErrorFrom(err)
	if !ok {
		return false
	}
	if respErr.Status != 200 {
		return false
	}
	if len(respErr.Errors) == 0 {
		return false
	}
	for _, msg := range respErr.Errors {
		if !strings.Contains(msg, shopifyInternalErrorMessage) {
			return false
		}
	}
	return true
}

func responseErrorFrom(err error) (goshopify.ResponseError, bool) {
	var respErr goshopify.ResponseError
	if errors.As(err, &respErr) {
		return respErr, true
	}

	var respErrPtr *goshopify.ResponseError
	if errors.As(err, &respErrPtr) && respErrPtr != nil {
		return *respErrPtr, true
	}

	return goshopify.ResponseError{}, false
}

// retryGraphQLRead retries idempotent read-only GraphQL queries on transient
// internal errors. The attempt must reset its response destination before each
// query. Do not use for mutations — they are not safe to retry.
func retryGraphQLRead(ctx context.Context, delays []time.Duration, attempt func() error) error {
	maxAttempts := len(delays) + 1
	var lastErr error

	for i := 0; i < maxAttempts; i++ {
		if err := ctx.Err(); err != nil {
			if lastErr != nil {
				return errors.Join(err, lastErr)
			}
			return err
		}

		lastErr = attempt()
		if lastErr == nil {
			return nil
		}
		if !isTransientGraphQLInternalError(lastErr) {
			return lastErr
		}
		if i == maxAttempts-1 {
			break
		}

		timer := time.NewTimer(delays[i])
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return errors.Join(ctx.Err(), lastErr)
		case <-timer.C:
		}
	}

	return lastErr
}
