BINARY=terraform-provider-sap-developer-hub

.PHONY: build test unit-test acceptance-test fmt vet lint tidy docs clean

build:
	go build -o $(BINARY) .

test: unit-test

unit-test:
	go test -race -cover ./...

acceptance-test:
	TF_ACC=1 go test -v -timeout 60m ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy
	cd tools && go mod tidy

docs:
	cd tools && go build -o ../.bin/tfplugindocs github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
	./.bin/tfplugindocs generate --provider-name developerhub

clean:
	rm -f $(BINARY)
	rm -rf .bin dist
