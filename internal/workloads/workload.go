package workloads

import (
	corev1 "k8s.io/api/core/v1"
	kubevirtv1 "kubevirt.io/api/core/v1"

	"virtwork/internal/cloudinit"
	"virtwork/internal/config"
)

// CloudConfigOpts is re-exported from cloudinit for convenience.
type CloudConfigOpts = cloudinit.CloudConfigOpts

// WriteFile is re-exported from cloudinit for convenience.
type WriteFile = cloudinit.WriteFile

// Workload defines the contract for all workload types.
// Implementations are pure data producers — no I/O, no goroutines.
type Workload interface {
	// Name returns the workload identifier (e.g., "cpu", "memory", "disk").
	Name() string

	// CloudInitUserdata returns the cloud-init YAML for this workload.
	CloudInitUserdata() (string, error)

	// VMResources returns the CPU and memory requirements for each VM.
	VMResources() VMResourceSpec

	// ExtraVolumes returns additional KubeVirt volumes beyond the base containerDisk
	// and cloudInitNoCloud volumes. Returns nil if none needed.
	ExtraVolumes() []kubevirtv1.Volume

	// ExtraDisks returns additional KubeVirt disks beyond the base containerDisk
	// and cloudInitNoCloud disks. Returns nil if none needed.
	ExtraDisks() []kubevirtv1.Disk

	// DataVolumeTemplates returns CDI DataVolumeTemplateSpecs for persistent storage.
	// Returns nil if no data volumes needed.
	DataVolumeTemplates() []kubevirtv1.DataVolumeTemplateSpec

	// RequiresService returns true if this workload needs a K8s Service.
	RequiresService() bool

	// ServiceSpec returns the K8s Service definition, or nil if not needed.
	ServiceSpec() *corev1.Service

	// VMCount returns the number of VMs this workload requires.
	VMCount() int
}

// VMResourceSpec holds CPU and memory requirements for a VM.
type VMResourceSpec struct {
	CPUCores int
	Memory   string
}

// BaseWorkload provides default implementations for optional Workload methods.
// Embed this struct in concrete workloads to inherit sensible defaults.
type BaseWorkload struct {
	Config            config.WorkloadConfig
	SSHUser           string
	SSHPassword       string
	SSHAuthorizedKeys []string
}

// VMResources returns the CPU and memory spec from the workload config.
func (b *BaseWorkload) VMResources() VMResourceSpec {
	return VMResourceSpec{
		CPUCores: b.Config.CPUCores,
		Memory:   b.Config.Memory,
	}
}

// ExtraVolumes returns nil — no additional volumes by default.
func (b *BaseWorkload) ExtraVolumes() []kubevirtv1.Volume {
	return nil
}

// ExtraDisks returns nil — no additional disks by default.
func (b *BaseWorkload) ExtraDisks() []kubevirtv1.Disk {
	return nil
}

// DataVolumeTemplates returns nil — no data volumes by default.
func (b *BaseWorkload) DataVolumeTemplates() []kubevirtv1.DataVolumeTemplateSpec {
	return nil
}

// RequiresService returns false — no K8s Service by default.
func (b *BaseWorkload) RequiresService() bool {
	return false
}

// ServiceSpec returns nil — no Service definition by default.
func (b *BaseWorkload) ServiceSpec() *corev1.Service {
	return nil
}

// VMCount returns 1 — single VM by default.
func (b *BaseWorkload) VMCount() int {
	return 1
}

// BuildCloudConfig injects SSH credentials into the given options and delegates
// to cloudinit.BuildCloudConfig. Workloads should call this instead of the
// package-level function to ensure consistent SSH credential handling.
func (b *BaseWorkload) BuildCloudConfig(opts CloudConfigOpts) (string, error) {
	opts.SSHUser = b.SSHUser
	opts.SSHPassword = b.SSHPassword
	opts.SSHAuthorizedKeys = b.SSHAuthorizedKeys
	return cloudinit.BuildCloudConfig(opts)
}
