FROM golang:1.27.1-trixie

# OpenTofu — pinned to match CI (opentofu/setup-opentofu@v2 in test.yml).
# The plugin-testing framework shells out to whatever binary is on PATH named
# "tofu"; terraform-plugin-testing detects it automatically.
ARG TOFU_VERSION=1.12.6
RUN apt-get update && apt-get install -y --no-install-recommends unzip \
 && ARCH=$(dpkg --print-architecture) \
 && curl -fsSL \
      "https://github.com/opentofu/opentofu/releases/download/v${TOFU_VERSION}/tofu_${TOFU_VERSION}_linux_${ARCH}.zip" \
      -o /tmp/tofu.zip \
 && unzip /tmp/tofu.zip -d /usr/local/bin tofu \
 && rm /tmp/tofu.zip \
 && tofu version \
 && apt-get clean && rm -rf /var/lib/apt/lists/*

# golangci-lint — pinned; never use the curl-pipe-sh install in containers.
ARG GOLANGCI_VERSION=2.13.2
RUN ARCH=$(dpkg --print-architecture) \
 && curl -fsSL \
      "https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_VERSION}/golangci-lint-${GOLANGCI_VERSION}-linux-${ARCH}.tar.gz" \
    | tar -xz -C /usr/local/bin --strip-components=1 \
         "golangci-lint-${GOLANGCI_VERSION}-linux-${ARCH}/golangci-lint" \
 && golangci-lint version

# Pre-populate the module cache so the first `make build` inside the container
# does not need to hit the network. Only the module files are needed here.
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download

# tfplugindocs is declared as a `tool` directive in go.mod; build it once so
# `make generate` works without network access inside the container.
RUN go tool tfplugindocs --help > /dev/null 2>&1 || true
