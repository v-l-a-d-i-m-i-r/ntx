set dotenv-load := true

# Show available recipes
default:
  @just --list

# Initialize the project
init:
  @[ -f .env ] || cp example.env .env;
  @go mod tidy;
  @git config core.hooksPath .githooks;

# Run all tests
test:
  @go test ./...;

# Run linter
lint:
  @go tool golangci-lint run;

# Run linter and fix
lint-fix:
  @go tool golangci-lint run --fix;

# Debug a specific linter (e.g., just debug-linter revive)
debug-linter linter_name:
  @GL_DEBUG={{linter_name}} go tool golangci-lint run --enable-only={{linter_name}} 2>&1;

# Build the ntx-cli binary
build-cli:
  @CGO_ENABLED=0 go build -ldflags="-s -w" -o ./bin/ntx-cli ./cmd/ntx-cli;

# Run the uuidv7 command directly
run-cli:
  @go run -race ./cmd/ntx-cli;

# Install the uuidv7 binary to the specified install directory
install-cli: build-cli
  @mkdir -p $INSTALL_DIR
  @cp ./bin/ntx-cli $INSTALL_DIR/ntx-cli;

# Build all binaries
build-all:
  @CGO_ENABLED=0 go build -ldflags="-s -w" -o ./bin/ \
    ./cmd/ntx-cli \
    ./cmd/ntx-backend;

# Install all binaries
install-all: build-all
  @mkdir -p $INSTALL_DIR
  @cp ./bin/ntx-cli $INSTALL_DIR/ntx-cli;
  @cp ./bin/ntx-backend $INSTALL_DIR/ntx-backend;
