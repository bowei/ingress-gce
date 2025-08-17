# Gemini Code Assistant Context

## Project Overview

This project is the Google Cloud Load Balancer (GCLB) controller for Kubernetes, also known as `ingress-gce`. It is responsible for managing GCE L7 load balancers based on Ingress resources defined in a Kubernetes cluster.

The controller is written in Go and utilizes the Kubernetes client-go library to interact with the Kubernetes API server. It watches for changes to Ingress, Service, and other related resources, and then creates and configures the necessary GCE networking resources (e.g., forwarding rules, target proxies, backend services) to expose services to external traffic.

The project also includes several auxiliary binaries, such as a 404-server for default backend functionality and an e2e-test binary for end-to-end testing.

## Building and Running

The project uses a `Makefile` to automate the build, test, and containerization process. The build process is containerized and uses a `golang:1.22.4` Docker image.

### Key Commands

*   **Build all binaries:**
    ```bash
    make build
    ```
    or for a specific architecture:
    ```bash
    make build-amd64
    ```

*   **Run unit tests and linters:**
    ```bash
    make test
    ```

*   **Build container images:**
    ```bash
    make containers
    ```

*   **Push container images to the registry:**
    ```bash
    make push
    ```
    The container image registry can be configured by setting the `REGISTRY` environment variable. The default is `gcr.io/k8s-image-staging`.

*   **Generate code:**
    ```bash
    make generate
    ```

*   **Run verification scripts:**
    ```bash
    make verify
    ```

## Development Conventions

### Coding Style

The project follows the standard Go coding style. The `make test` command runs `gofmt` and `golangci-lint` to enforce code formatting and style.

### Testing

Unit tests are located in the same packages as the code they test and are run with the `go test` command. The `make test` command runs all unit tests.

End-to-end tests are located in the `cmd/e2e-test` directory.

### Dependencies

The project uses Go modules to manage dependencies. The `go.mod` file lists the project's dependencies.
