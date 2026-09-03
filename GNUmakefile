default: build

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	go generate ./...

fmt:
	gofmt -s -w .

test:
	go test -v -count=1 -parallel=4 ./...

testacc:
	TF_ACC=1 go test -v -count=1 -parallel=4 -timeout 10m ./...

# Archive leftover tf-acc-* objects in the Sandbox. Run only when no acceptance job is active.
sweep:
	go test ./internal/provider/ -v -sweep=all -timeout 10m

.PHONY: build install lint generate fmt test testacc sweep
