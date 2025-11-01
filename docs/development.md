# Development Guide

This document describes how to build, test, and run the Bot Network Policy Operator locally.

## Prerequisites

- Go 1.21+
- Access to the Kubernetes Go client and controller-runtime modules (the project uses Go modules).
- (Optional) [`kubebuilder`](https://book.kubebuilder.io/) or [`controller-gen`](https://github.com/kubernetes-sigs/controller-tools) if you plan to regenerate CRDs and RBAC manifests.

## Setup

```bash
# download dependencies
GOPROXY=https://proxy.golang.org,direct go mod download

# run unit tests
go test ./...

# format code
gofmt -w ./
```

## Running the Operator Locally

1. Ensure you have a valid kubeconfig pointing at a development cluster.
2. Install the CRD:

   ```bash
   controller-gen crd paths=./api/... output:crd:dir=config/crd/bases
   kubectl apply -f config/crd/bases
   ```

3. Run the manager:

   ```bash
   go run ./cmd/operator
   ```

4. Apply an example `BotNetworkPolicy` resource to watch the controller reconcile.

## Docker Image

Use the provided `Dockerfile` to build a container image:

```bash
docker build -t ghcr.io/sugaf1204/botnetworkpolicy-operator:dev .
```

## Directory Structure

- `api/v1alpha1/`: CRD type definitions.
- `cmd/operator/`: Main entry point for running the controller manager.
- `pkg/controllers/`: Controller reconciliation logic and unit tests.
- `pkg/providers/`: Provider implementations that fetch CIDRs from various sources.
- `docs/`: Project documentation.

## Adding Providers

To add a new provider:

1. Update `api/v1alpha1/botnetworkpolicy_types.go` with any additional configuration fields.
2. Implement the fetcher under `pkg/providers/` and register it in `Factory.FromSpec`.
3. Add unit tests covering the new provider logic.
4. Document usage in the README or docs as appropriate.
