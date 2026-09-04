package shopify

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	goshopify "github.com/bold-commerce/go-shopify/v4"
)

type sequentialGraphQLTransport struct {
	responses []string
	calls     int
}

func (t *sequentialGraphQLTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.calls >= len(t.responses) {
		return nil, fmt.Errorf("unexpected GraphQL request %d", t.calls+1)
	}

	body := t.responses[t.calls]
	t.calls++

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Request:    req,
	}, nil
}

func TestGetMetaobjectDefinition_RetriesTransientInternalError(t *testing.T) {
	const internalErrMsg = shopifyInternalErrorMessage + " Request ID: req-1"
	partialResponse := `{
  "data": {
    "metaobjectDefinition": {
      "id": "gid://shopify/MetaobjectDefinition/1",
      "type": "test_type",
      "name": "stale",
      "description": "stale-desc",
      "fieldDefinitions": [],
      "hasThumbnailField": false
    }
  },
  "errors": [
    {"message": "` + internalErrMsg + `"}
  ]
}`
	successResponse := `{
  "data": {
    "metaobjectDefinition": {
      "id": "gid://shopify/MetaobjectDefinition/1",
      "type": "test_type",
      "name": "fresh",
      "fieldDefinitions": [],
      "hasThumbnailField": false
    }
  }
}`

	transport := &sequentialGraphQLTransport{
		responses: []string{partialResponse, successResponse},
	}
	httpClient := &http.Client{Transport: transport}
	rawClient, err := goshopify.NewClient(
		goshopify.App{},
		"testshop",
		"token",
		goshopify.WithHTTPClient(httpClient),
		goshopify.WithVersion("2024-07"),
	)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	client := NewClient(rawClient)
	client.graphQLReadRetryDelays = zeroDelays(4)
	def, err := client.GetMetaobjectDefinition(context.Background(), "gid://shopify/MetaobjectDefinition/1")
	if err != nil {
		t.Fatalf("GetMetaobjectDefinition returned error: %v", err)
	}
	if transport.calls != 2 {
		t.Fatalf("expected 2 GraphQL requests, got %d", transport.calls)
	}
	if def == nil {
		t.Fatal("expected metaobject definition, got nil")
	}
	if def.Name != "fresh" {
		t.Fatalf("expected name %q, got %q", "fresh", def.Name)
	}
	if def.Description != "" {
		t.Fatalf("expected zeroed description, got %q", def.Description)
	}
}
