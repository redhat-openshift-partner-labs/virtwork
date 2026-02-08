# virtwork

virtwork is a CLI tool that creates virtual machines on OpenShift clusters with [OpenShift Virtualization](https://docs.openshift.com/container-platform/latest/virt/about_virt/about-virt.html) (CNV) installed and runs continuous workloads inside them. It produces realistic CPU, memory, database, network, and disk I/O metrics for monitoring systems like Prometheus and Grafana.

virtwork is a **one-shot deployment tool** — it creates resources and exits. Workload lifecycle management is handled by systemd inside each VM.

## Prerequisites

- Go 1.25+
- An OpenShift cluster with OpenShift Virtualization (KubeVirt/CNV) installed
- `kubeconfig` access to the cluster (or running in-cluster)

## Installation

```bash
go build -o virtwork ./cmd/virtwork
```

Or run directly:

```bash
go run ./cmd/virtwork --help
```

## Quick Start

```bash
# Preview what would be deployed (no cluster required)
virtwork run --dry-run

# Deploy all workloads with defaults
virtwork run

# Deploy specific workloads
virtwork run --workloads cpu,memory,disk

# Deploy with SSH access for debugging
virtwork run --ssh-user virtwork --ssh-key-file ~/.ssh/id_ed25519.pub

# Clean up all managed resources
virtwork cleanup
```

## Workloads

| Workload | VMs | Description | Tools |
|----------|-----|-------------|-------|
| **cpu** | N (configurable) | Continuous CPU stress | `stress-ng --cpu 0 --cpu-method all` |
| **memory** | N (configurable) | Memory pressure at 80% | `stress-ng --vm 1 --vm-bytes 80%` |
| **database** | N (configurable) | PostgreSQL with pgbench loop | `pgbench -c 10 -j 2 -T 300` |
| **network** | 2 (server + client) | Bidirectional throughput | `iperf3 --bidir` |
| **disk** | N (configurable) | Mixed random and sequential I/O | `fio` with multiple profiles |

All workloads run as systemd services inside the VMs, surviving reboots and auto-restarting on failure.

## Usage

### `virtwork run`

Deploy VMs with workloads.

```
Flags:
      --workloads strings          Workloads to deploy (default [cpu,database,disk,memory,network])
      --vm-count int               Number of VMs per workload (default 1)
      --cpu-cores int              CPU cores per VM
      --memory string              Memory per VM (e.g., 2Gi)
      --disk-size string           Data disk size
      --container-disk-image string Container disk image for VMs
      --dry-run                    Print specs without creating resources
      --no-wait                    Skip waiting for VM readiness
      --timeout int                Readiness timeout in seconds
      --ssh-user string            SSH user for VMs
      --ssh-password string        SSH password for VMs
      --ssh-key strings            SSH authorized key (repeatable)
      --ssh-key-file strings       SSH key file path (repeatable)

Global Flags:
      --namespace string           Kubernetes namespace for VMs
      --kubeconfig string          Path to kubeconfig file
      --config string              Path to YAML config file
      --verbose                    Enable verbose output
```

### `virtwork cleanup`

Delete all resources managed by virtwork.

```
Flags:
      --delete-namespace           Also delete the namespace
```

Cleanup is error-tolerant — individual resource deletion failures are logged but do not abort the operation. All resources are tracked via the `app.kubernetes.io/managed-by: virtwork` label, so cleanup works even if the tool crashed mid-deployment.

## Configuration

virtwork uses a priority chain for configuration (highest to lowest):

1. CLI flags
2. Environment variables (`VIRTWORK_` prefix)
3. YAML config file (`--config`)
4. Defaults

### Environment Variables

| Variable | Description |
|----------|-------------|
| `VIRTWORK_NAMESPACE` | Kubernetes namespace |
| `VIRTWORK_SSH_USER` | SSH user for VMs |
| `VIRTWORK_SSH_PASSWORD` | SSH password for VMs |
| `VIRTWORK_SSH_AUTHORIZED_KEYS` | Comma-separated SSH public keys |

### YAML Config File

```yaml
namespace: virtwork-prod
container_disk_image: quay.io/containerdisks/fedora:41
data_disk_size: 20Gi

ssh_user: virtwork
ssh_authorized_keys:
  - ssh-ed25519 AAAA...

workloads:
  cpu:
    enabled: true
    vm_count: 2
    cpu_cores: 4
    memory: 4Gi
  database:
    enabled: true
    cpu_cores: 2
    memory: 4Gi
```

## SSH Access

VMs can be configured with SSH access for debugging and inspection.

```bash
# Deploy with SSH key
virtwork run --ssh-user virtwork --ssh-key-file ~/.ssh/id_ed25519.pub

# Access via virtctl
virtctl ssh --ssh-key ~/.ssh/id_ed25519 virtwork@virtwork-cpu-0

# Access via port forward
oc port-forward vmi/virtwork-cpu-0 2222:22
ssh -p 2222 virtwork@localhost
```

When no SSH flags are provided, no user account is configured in the VMs.

> **Note:** SSH passwords passed via `--ssh-password` are visible in process listings and stored as plaintext in the VM spec. Use SSH key authentication for anything beyond test environments.

## Architecture

The codebase follows a strict layered architecture where each layer depends only on layers below it.

```
Layer 4 — Orchestration     cmd/virtwork, cleanup
Layer 3 — Workload Defs     workloads (interface, cpu, memory, database, network, disk, registry)
Layer 2 — K8s Abstractions  vm, resources, wait
Layer 1 — Infrastructure    config, cluster, cloudinit
Layer 0 — Definitions       constants
```

Concurrency uses goroutines with `errgroup.Group` for structured error handling and `context.Context` for timeouts and cancellation. VM creation, readiness polling, and cleanup all run concurrently.

See [docs/architecture.md](docs/architecture.md) for detailed diagrams and design decisions.

## Project Structure

```
virtwork/
├── cmd/virtwork/main.go           # Cobra CLI + orchestration
├── internal/
│   ├── constants/                 # API coordinates, labels, defaults
│   ├── config/                    # Viper-based config priority chain
│   ├── cluster/                   # controller-runtime client init
│   ├── cloudinit/                 # Cloud-config YAML builder
│   ├── vm/                        # VM spec construction + CRUD + retry
│   ├── resources/                 # Namespace + Service helpers
│   ├── wait/                      # VMI readiness polling
│   ├── cleanup/                   # Label-based teardown
│   └── workloads/                 # Workload interface + 5 implementations + registry
├── tests/
│   ├── integration/               # Integration tests (requires cluster)
│   └── e2e/                       # E2E tests (requires cluster)
├── docs/
│   ├── architecture.md            # Layered architecture and diagrams
│   ├── development.md             # Developer guide
│   └── implementation-plan.md     # Phased build plan
├── go.mod
└── go.sum
```

## Development

### Testing

```bash
# Unit tests
go test ./...

# With race detector
go test -race ./...

# Using Ginkgo BDD runner
ginkgo -r

# Integration tests (requires cluster)
go test -tags integration ./tests/integration/...

# E2E tests (requires cluster)
go test -tags e2e ./tests/e2e/...
```

### Building

```bash
go build -o virtwork ./cmd/virtwork
```

See [docs/development.md](docs/development.md) for the full developer guide, including instructions for adding new workloads.

## License

Apache License 2.0. See [LICENSE](LICENSE).
