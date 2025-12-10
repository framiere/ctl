# Conduktor CLI (ctl)

## Project Overview

**Conduktor CLI** is a command-line tool designed to interact with the Conduktor Console. It draws inspiration from the Kubernetes `kubectl` CLI, employing similar concepts and usage patterns.

*   **Purpose:** Manage Conduktor resources (apply, get, delete, edit) and handle authentication.
*   **Language:** Go (requires version 1.22+).
*   **Key Concepts:** Uses a `kind` based resource management system (similar to Kubernetes `apiVersion` and `kind`).

## Building and Running

The project includes a `Makefile` to streamline common development tasks.

### Prerequisites
*   Go 1.22+
*   Docker (for running the image or integration tests)

### Build
To build the binary (output as `conduktor` in the root):
```bash
make build
# OR
go build -o conduktor .
```

### Run
To run the CLI directly from source, you need to set up the environment variables:
```bash
export CDK_BASE_URL=http://localhost:8080
export CDK_API_KEY=<your_admin_token>
go run . <command>
```

### Testing
*   **Run all tests (Unit + Integration):**
    ```bash
    make test
    ```
*   **Run Unit Tests only:**
    ```bash
    go test ./...
    ```
*   **Run Integration Tests only:**
    ```bash
    ./scripts/test_final_exec.sh
    ```

### Linting and Formatting
*   **Format code:**
    ```bash
    make fmt
    ```
*   **Lint code:**
    ```bash
    make lint
    ```

### Setup
*   **Install development dependencies:**
    ```bash
    make install
    ```
*   **Setup Git hooks (pre-commit, detect-secrets, golangci-lint):**
    ```bash
    make setup-hooks
    ```

## Development Conventions

*   **Commit Messages:** Must follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification (e.g., `feat: ...`, `fix: ...`).
*   **Versioning:** All versions start with `v` (e.g., `v0.2.6`) to satisfy Go module requirements.
*   **Configuration:** The CLI relies heavily on environment variables for configuration (Base URL, Auth tokens, TLS certs).
    *   `CDK_BASE_URL`: URL of the Conduktor Console.
    *   `CDK_API_KEY`: API Access Token.
    *   `CDK_INSECURE=true`: Disable TLS verification.
    *   See `README.md` for Gateway and Teleport configuration details.
