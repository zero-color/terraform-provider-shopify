package shopify

import (
	"time"

	goshopify "github.com/bold-commerce/go-shopify/v4"
)

type Client struct {
	shopifyClient          *goshopify.Client
	graphQLReadRetryDelays []time.Duration
}

func NewClient(shopifyClient *goshopify.Client) *Client {
	return &Client{
		shopifyClient:          shopifyClient,
		graphQLReadRetryDelays: defaultGraphQLReadRetryDelays,
	}
}
