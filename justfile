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

# Generate a new internal/db migration file (e.g. just generate-migration create_notifications)
generate-migration slug:
  @./tools/new-migration.sh {{slug}};

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

# Build the ntx-dbus-forwarder binary
build-dbus-forwarder:
  @CGO_ENABLED=0 go build -ldflags="-s -w" -o ./bin/ntx-dbus-forwarder ./cmd/ntx-dbus-forwarder;

# Run the ntx-dbus-forwarder daemon directly
run-dbus-forwarder: build-dbus-forwarder
  @./bin/ntx-dbus-forwarder;

# Install the ntx-dbus-forwarder binary to the specified install directory
install-dbus-forwarder: build-dbus-forwarder
  @mkdir -p $INSTALL_DIR
  @cp ./bin/ntx-dbus-forwarder $INSTALL_DIR/ntx-dbus-forwarder;

# Build the ntx-server binary
build-server:
  @CGO_ENABLED=0 go build -ldflags="-s -w" -o ./bin/ntx-server ./cmd/ntx-server;

# Run the ntx-server daemon directly
run-server: build-server
  @./bin/ntx-server;

# Install the ntx-server binary to the specified install directory
install-server: build-server
  @mkdir -p $INSTALL_DIR
  @cp ./bin/ntx-server $INSTALL_DIR/ntx-server;

# Build all binaries
build-all:
  @CGO_ENABLED=0 go build -ldflags="-s -w" -o ./bin/ \
    ./cmd/ntx-cli \
    ./cmd/ntx-server \
    ./cmd/ntx-dbus-forwarder;

# Install all binaries
install-all: build-all
  @mkdir -p $INSTALL_DIR
  @cp ./bin/ntx-cli $INSTALL_DIR/ntx-cli;
  @cp ./bin/ntx-server $INSTALL_DIR/ntx-server;
  @cp ./bin/ntx-dbus-forwarder $INSTALL_DIR/ntx-dbus-forwarder;
