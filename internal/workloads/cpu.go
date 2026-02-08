// Copyright 2026 Red Hat
// SPDX-License-Identifier: Apache-2.0

package workloads

import (
	"virtwork/internal/config"
)

const cpuSystemdUnit = `[Unit]
Description=Virtwork CPU stress workload
After=network.target

[Service]
Type=simple
ExecStart=/usr/bin/stress-ng --cpu 0 --cpu-method all --timeout 0
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
`

// CPUWorkload generates cloud-init userdata for a continuous CPU stress workload
// using stress-ng.
type CPUWorkload struct {
	BaseWorkload
}

// NewCPUWorkload creates a CPUWorkload with the given configuration and SSH credentials.
func NewCPUWorkload(cfg config.WorkloadConfig, sshUser, sshPassword string, sshKeys []string) *CPUWorkload {
	return &CPUWorkload{
		BaseWorkload: BaseWorkload{
			Config:            cfg,
			SSHUser:           sshUser,
			SSHPassword:       sshPassword,
			SSHAuthorizedKeys: sshKeys,
		},
	}
}

// Name returns "cpu".
func (w *CPUWorkload) Name() string {
	return "cpu"
}

// CloudInitUserdata returns cloud-init YAML that installs stress-ng and runs a
// continuous CPU stress workload via systemd.
func (w *CPUWorkload) CloudInitUserdata() (string, error) {
	return w.BuildCloudConfig(CloudConfigOpts{
		Packages: []string{"stress-ng"},
		WriteFiles: []WriteFile{
			{
				Path:        "/etc/systemd/system/virtwork-cpu.service",
				Content:     cpuSystemdUnit,
				Permissions: "0644",
			},
		},
		RunCmd: [][]string{
			{"systemctl", "daemon-reload"},
			{"systemctl", "enable", "--now", "virtwork-cpu.service"},
		},
	})
}
