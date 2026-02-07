package workloads_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"virtwork/internal/config"
	"virtwork/internal/workloads"
)

var _ = Describe("DiskWorkload", func() {
	var w *workloads.DiskWorkload

	BeforeEach(func() {
		w = workloads.NewDiskWorkload(config.WorkloadConfig{
			Enabled:  true,
			VMCount:  1,
			CPUCores: 2,
			Memory:   "2Gi",
		}, "10Gi", "virtwork", "", nil)
	})

	It("should return 'disk' for Name", func() {
		Expect(w.Name()).To(Equal("disk"))
	})

	It("should include fio in packages", func() {
		result, err := w.CloudInitUserdata()
		Expect(err).NotTo(HaveOccurred())

		parsed := parseYAML(result)
		pkgs, ok := parsed["packages"].([]interface{})
		Expect(ok).To(BeTrue())
		Expect(pkgs).To(ContainElement("fio"))
	})

	It("should include fio profiles in write_files", func() {
		result, err := w.CloudInitUserdata()
		Expect(err).NotTo(HaveOccurred())

		parsed := parseYAML(result)
		files := parsed["write_files"].([]interface{})

		// Should have: mixed-rw.fio, seq-write.fio, systemd unit = 3 files
		Expect(files).To(HaveLen(3))

		paths := make([]string, len(files))
		for i, f := range files {
			paths[i] = f.(map[string]interface{})["path"].(string)
		}
		Expect(paths).To(ContainElement("/etc/fio/mixed-rw.fio"))
		Expect(paths).To(ContainElement("/etc/fio/seq-write.fio"))
		Expect(paths).To(ContainElement("/etc/systemd/system/virtwork-disk.service"))
	})

	It("should have data volume template", func() {
		dvts := w.DataVolumeTemplates()
		Expect(dvts).To(HaveLen(1))
		Expect(dvts[0].Name).To(Equal("virtwork-disk-data"))
	})

	It("should have extra disk for data volume", func() {
		disks := w.ExtraDisks()
		Expect(disks).To(HaveLen(1))
		Expect(disks[0].Name).To(Equal("datadisk"))

		volumes := w.ExtraVolumes()
		Expect(volumes).To(HaveLen(1))
		Expect(volumes[0].Name).To(Equal("datadisk"))
	})

	It("should not require service", func() {
		Expect(w.RequiresService()).To(BeFalse())
		Expect(w.ServiceSpec()).To(BeNil())
	})
})
