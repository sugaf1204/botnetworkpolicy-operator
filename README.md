# Bot Network Policy Operator

[![CI](https://github.com/sugaf1204/botnetworkpolicy-operator/actions/workflows/ci.yaml/badge.svg)](https://github.com/sugaf1204/botnetworkpolicy-operator/actions/workflows/ci.yaml)
[![CD](https://github.com/sugaf1204/botnetworkpolicy-operator/actions/workflows/cd.yaml/badge.svg)](https://github.com/sugaf1204/botnetworkpolicy-operator/actions/workflows/cd.yaml)

The Bot Network Policy Operator manages Kubernetes `NetworkPolicy` objects that allow ingress/egress traffic exclusively from known bot IP ranges published by popular cloud platforms and custom sources.  It periodically refreshes provider allowlists and keeps a deterministic NetworkPolicy in sync for each `BotNetworkPolicy` custom resource.

## Features

- Built-in providers for Google, AWS, and GitHub bot/metadata endpoints.
- ConfigMap provider to supply custom CIDR ranges managed within the cluster.
- JSON endpoint provider that retrieves CIDRs from an arbitrary HTTP endpoint and extracts them via a JSON field path.
- Deterministic NetworkPolicy generation with optional ingress/egress toggles and custom CIDR overrides.
- Periodic re-sync with configurable intervals per resource.

## Custom Resource Overview

```yaml
apiVersion: bot.networking.dev/v1alpha1
kind: BotNetworkPolicy
metadata:
  name: example
spec:
  podSelector:
    matchLabels:
      app: web
  syncPeriod: 30m
  providers:
    - name: google
    - name: aws
    - name: github
    - name: configMap
      configMap:
        name: extra-bot-ips
        key: cidrs
    - name: jsonEndpoint
      jsonEndpoint:
        url: https://example.com/bots.json
        fieldPath: data.cidrs
        headers:
          Accept: application/json
        headerSecretRefs:
          - name: Authorization
            secretKeyRef:
              name: bot-endpoint-token
              key: token
  customCidrs:
    - 192.0.2.0/24
```

The operator will create or update a `NetworkPolicy` named `<metadata.name>-allow-bots` (or a custom name specified via the `bot.networking.dev/networkpolicy-name` annotation) in the same namespace. The generated policy contains ingress rules (and optional egress rules) limited to the merged set of CIDRs.

## Installation

### Using Helm

The operator can be easily installed using Helm. The Helm chart includes the CRD (CustomResourceDefinition) in the `crds/` directory, which will be automatically installed by Helm.

```bash
# Install from local Helm chart (for development)
helm install botnetworkpolicy-operator ./charts/botnetworkpolicy-operator-chart \
  --namespace botnetworkpolicy-system \
  --create-namespace

# Or install from GitHub Container Registry (after first release)
helm install botnetworkpolicy-operator \
  oci://ghcr.io/sugaf1204/chart/botnetworkpolicy-operator-chart \
  --version 0.1.0 \
  --namespace botnetworkpolicy-system \
  --create-namespace
```

**Note about CRDs**: Helm automatically installs CRDs placed in the `crds/` directory during installation. However, Helm does **not** upgrade or delete CRDs during chart upgrades or uninstalls. If you need to update the CRD manually:

```bash
# Apply the CRD directly
kubectl apply -f charts/botnetworkpolicy-operator-chart/crds/bot.networking.dev_botnetworkpolicies.yaml

# Or if upgrading from OCI registry, extract and apply
helm pull oci://ghcr.io/sugaf1204/chart/botnetworkpolicy-operator-chart --version 0.1.0 --untar
kubectl apply -f botnetworkpolicy-operator-chart/crds/
```

### Configuration

You can customize the installation by overriding values:

```bash
helm install botnetworkpolicy-operator ./charts/botnetworkpolicy-operator-chart \
  --namespace botnetworkpolicy-system \
  --create-namespace \
  --set image.tag=v0.1.0 \
  --set replicaCount=2 \
  --set resources.limits.memory=512Mi
```

See [values.yaml](charts/botnetworkpolicy-operator-chart/values.yaml) for all available configuration options.

## Getting Started

1. Install the operator using Helm (see Installation section above).
2. Apply a `BotNetworkPolicy` resource in the target namespace.
3. Confirm that a `NetworkPolicy` with the `botnetworkpolicy.bot.networking.dev/owner` label appears and contains the expected IP blocks.

See [`docs/development.md`](docs/development.md) for development workflows and testing guidance.

## CI/CD

This project uses GitHub Actions for continuous integration and deployment:

- **CI Pipeline** (`.github/workflows/ci.yaml`): Runs on every push and pull request
  - Executes Go tests with race detection and coverage
  - Runs golangci-lint for code quality
  - Builds Docker image to verify successful build
  - Lints and validates Helm chart

- **CD Pipeline** (`.github/workflows/cd.yaml`): Runs on version tags (e.g., `v1.0.0`)
  - Builds multi-architecture Docker images (amd64, arm64)
  - Pushes images to GitHub Container Registry (`ghcr.io`)
  - Packages and pushes Helm chart to GitHub Container Registry (`ghcr.io`)
  - Releases Helm chart as a GitHub release asset

### Creating a Release

To create a new release:

```bash
# Create and push a version tag
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

The CD pipeline will automatically:
1. Build and push the Docker image with tags: `v0.1.0`, `v0.1`, `v0`
2. Package the Helm chart with version `0.1.0`
3. Create a GitHub release with the Helm chart package
