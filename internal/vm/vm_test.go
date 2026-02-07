package vm_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	kubevirtv1 "kubevirt.io/api/core/v1"

	"virtwork/internal/vm"
)

var _ = Describe("BuildVMSpec", func() {
	var (
		opts   vm.VMSpecOpts
		result *kubevirtv1.VirtualMachine
	)

	BeforeEach(func() {
		opts = vm.VMSpecOpts{
			Name:               "test-vm",
			Namespace:          "test-ns",
			ContainerDiskImage: "quay.io/containerdisks/fedora:41",
			CloudInitUserdata:  "#cloud-config\n",
			CPUCores:           2,
			Memory:             "2Gi",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "virtwork",
				"app.kubernetes.io/name":       "cpu",
			},
		}
		result = vm.BuildVMSpec(opts)
	})

	It("should set correct API version and kind", func() {
		Expect(result.APIVersion).To(Equal("kubevirt.io/v1"))
		Expect(result.Kind).To(Equal("VirtualMachine"))
	})

	It("should set name and namespace", func() {
		Expect(result.Name).To(Equal("test-vm"))
		Expect(result.Namespace).To(Equal("test-ns"))
	})

	It("should configure containerDisk volume", func() {
		volumes := result.Spec.Template.Spec.Volumes
		var containerDisk *kubevirtv1.Volume
		for i := range volumes {
			if volumes[i].Name == "containerdisk" {
				containerDisk = &volumes[i]
				break
			}
		}
		Expect(containerDisk).NotTo(BeNil())
		Expect(containerDisk.ContainerDisk).NotTo(BeNil())
		Expect(containerDisk.ContainerDisk.Image).To(Equal("quay.io/containerdisks/fedora:41"))
	})

	It("should configure cloudInitNoCloud volume", func() {
		volumes := result.Spec.Template.Spec.Volumes
		var cloudInit *kubevirtv1.Volume
		for i := range volumes {
			if volumes[i].Name == "cloudinitdisk" {
				cloudInit = &volumes[i]
				break
			}
		}
		Expect(cloudInit).NotTo(BeNil())
		Expect(cloudInit.CloudInitNoCloud).NotTo(BeNil())
		Expect(cloudInit.CloudInitNoCloud.UserData).To(Equal("#cloud-config\n"))
	})

	It("should set labels", func() {
		Expect(result.Labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "virtwork"))
		Expect(result.Labels).To(HaveKeyWithValue("app.kubernetes.io/name", "cpu"))
		// Labels should also propagate to the template
		Expect(result.Spec.Template.ObjectMeta.Labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "virtwork"))
	})

	It("should set CPU and memory resources", func() {
		domain := result.Spec.Template.Spec.Domain
		Expect(domain.CPU).NotTo(BeNil())
		Expect(domain.CPU.Cores).To(Equal(uint32(2)))
		memReq := domain.Resources.Requests[corev1.ResourceMemory]
		expected := resource.MustParse("2Gi")
		Expect(memReq.Equal(expected)).To(BeTrue())
	})

	It("should set running to true", func() {
		Expect(result.Spec.Running).NotTo(BeNil())
		Expect(*result.Spec.Running).To(BeTrue())
	})

	It("should configure masquerade networking", func() {
		interfaces := result.Spec.Template.Spec.Domain.Devices.Interfaces
		Expect(interfaces).To(HaveLen(1))
		Expect(interfaces[0].Name).To(Equal("default"))
		Expect(interfaces[0].Masquerade).NotTo(BeNil())

		networks := result.Spec.Template.Spec.Networks
		Expect(networks).To(HaveLen(1))
		Expect(networks[0].Name).To(Equal("default"))
		Expect(networks[0].Pod).NotTo(BeNil())
	})

	It("should include extra disks when provided", func() {
		opts.ExtraDisks = []kubevirtv1.Disk{
			{
				Name: "datadisk",
				DiskDevice: kubevirtv1.DiskDevice{
					Disk: &kubevirtv1.DiskTarget{
						Bus: "virtio",
					},
				},
			},
		}
		opts.ExtraVolumes = []kubevirtv1.Volume{
			{
				Name: "datadisk",
				VolumeSource: kubevirtv1.VolumeSource{
					DataVolume: &kubevirtv1.DataVolumeSource{
						Name: "test-data",
					},
				},
			},
		}
		result = vm.BuildVMSpec(opts)

		disks := result.Spec.Template.Spec.Domain.Devices.Disks
		Expect(disks).To(HaveLen(3)) // containerdisk + cloudinitdisk + datadisk

		volumes := result.Spec.Template.Spec.Volumes
		Expect(volumes).To(HaveLen(3))
	})

	It("should include data volume templates when provided", func() {
		dvt := vm.BuildDataVolumeTemplate("test-data", "10Gi")
		opts.DataVolumeTemplates = []kubevirtv1.DataVolumeTemplateSpec{dvt}
		result = vm.BuildVMSpec(opts)

		Expect(result.Spec.DataVolumeTemplates).To(HaveLen(1))
		Expect(result.Spec.DataVolumeTemplates[0].Name).To(Equal("test-data"))
	})
})

var _ = Describe("BuildDataVolumeTemplate", func() {
	It("should set name", func() {
		dvt := vm.BuildDataVolumeTemplate("data-disk", "20Gi")
		Expect(dvt.Name).To(Equal("data-disk"))
	})

	It("should set blank source", func() {
		dvt := vm.BuildDataVolumeTemplate("data-disk", "20Gi")
		Expect(dvt.Spec.Source).NotTo(BeNil())
		Expect(dvt.Spec.Source.Blank).NotTo(BeNil())
	})

	It("should set storage size", func() {
		dvt := vm.BuildDataVolumeTemplate("data-disk", "20Gi")
		Expect(dvt.Spec.Storage).NotTo(BeNil())
		storageReq := dvt.Spec.Storage.Resources.Requests[corev1.ResourceStorage]
		expected := resource.MustParse("20Gi")
		Expect(storageReq.Equal(expected)).To(BeTrue())
	})
})
