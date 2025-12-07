default: testacc

DEV_VERSION=999.999.999

# Build provider and place it plugins directory to be able to sideload the built provider.
.PHONY: install
install:
	go build -o ~/.terraform.d/plugins/registry.terraform.io/zero-clor/shopify/${DEV_VERSION}/darwin_arm64/terraform-provider-shopify

.PHONY: uninstall
uninstall:
	rm -r ~/.terraform.d/plugins/registry.terraform.io/zero-clor/shopify/${DEV_VERSION}

.PHONY: generate
generate:
	go generate ./...

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

.PHONY: lint
lint:
	@golangci-lint run

.PHONY: fmt
fmt:
	goimports -w .
	terraform fmt --recursive