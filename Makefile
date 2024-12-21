GOLANGCI_LINT = $(GOPATH)/bin/golangci-lint
GOLANGCI_LINT_VERSION = v1.62.2

$(GOLANGCI_LINT):
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin $(GOLANGCI_LINT_VERSION)

.PHONY: lint
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run
