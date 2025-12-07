# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Terraform provider for Shopify, built using the Terraform Plugin Framework. It manages Shopify resources via the Admin GraphQL API.

## Common Commands

```bash
# Build the provider
go install

# Run all tests (requires Shopify credentials)
TF_ACC=1 go test -v ./internal/provider/

# Run a specific test
TF_ACC=1 go test -v ./internal/provider/ -run TestAccMetafieldDefinitionResource

# Run linter
golangci-lint run

# Generate documentation (requires Terraform CLI)
go generate ./...
```

## Environment Variables for Testing

Acceptance tests require these environment variables:
- `SHOPIFY_SHOP` - Shop domain (e.g., `myshop.myshopify.com` or just `myshop`)
- `SHOPIFY_API_VERSION` - API version (e.g., `2024-07`)
- `SHOPIFY_API_KEY` - App API key
- `SHOPIFY_API_SECRET_KEY` - App API secret key
- `SHOPIFY_ADMIN_API_ACCESS_TOKEN` - Admin API access token

## Architecture

### Package Structure

- `internal/provider/` - Terraform provider implementation
  - `provider.go` - Provider configuration and client setup
  - `resource_*.go` - Resource implementations (CRUD operations)
  - `*_test.go` - Acceptance tests using terraform-plugin-testing

- `internal/shopify/` - Shopify API client wrapper
  - `client.go` - Wraps the go-shopify client
  - `*.go` - GraphQL operations for each resource type (mutations/queries)
  - `error.go` - Error handling for GraphQL UserErrors

- `internal/utils/` - Shared utilities (plan modifiers, HTTP debugging)

- `pkg/xslice/` - Generic slice utilities

### Key Dependencies

- `github.com/bold-commerce/go-shopify/v4` - Shopify API client (GraphQL)
- `github.com/hashicorp/terraform-plugin-framework` - Terraform Plugin Framework

### Resource Pattern

Each resource follows this pattern:
1. Resource struct in `internal/provider/resource_*.go` implements CRUD via Framework interfaces
2. GraphQL operations defined in `internal/shopify/*.go` with typed request/response structs
3. Shopify API responses include `userErrors` which are converted to Go errors via `error.go`

### Testing Pattern

Acceptance tests use `terraform-plugin-testing` and require a real Shopify store. Tests use `randResourceID()` to generate unique resource identifiers prefixed with `test_`.