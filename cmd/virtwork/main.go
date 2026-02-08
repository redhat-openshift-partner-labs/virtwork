// Copyright 2026 Red Hat
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	sigyaml "sigs.k8s.io/yaml"

	"virtwork/internal/cleanup"
	"virtwork/internal/cluster"
	"virtwork/internal/config"
	"virtwork/internal/constants"
	"virtwork/internal/resources"
	"virtwork/internal/vm"
	"virtwork/internal/wait"
	"virtwork/internal/workloads"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "virtwork",
		Short: "Create VMs on OpenShift with continuous workloads",
		Long: `Virtwork creates virtual machines on OpenShift clusters (with OpenShift
Virtualization installed) and runs continuous workloads inside them to produce
realistic CPU, memory, database, network, and disk I/O metrics.`,
		SilenceUsage: true,
	}

	pf := rootCmd.PersistentFlags()
	pf.String("namespace", "", "Kubernetes namespace for VMs")
	pf.String("kubeconfig", "", "Path to kubeconfig file")
	pf.String("config", "", "Path to YAML config file")
	pf.Bool("verbose", false, "Enable verbose output")

	rootCmd.AddCommand(newRunCmd(), newCleanupCmd())
	return rootCmd
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Create VMs and start workloads",
		Long: `Deploy virtual machines with the specified workloads. Each workload type
installs and configures its own software via cloud-init and runs continuously
via systemd.`,
		RunE: runE,
	}

	f := cmd.Flags()
	f.StringSlice("workloads", workloads.AllWorkloadNames, "Workloads to deploy (comma-separated)")
	f.Int("vm-count", 1, "Number of VMs per workload")
	f.Int("cpu-cores", 0, "CPU cores per VM")
	f.String("memory", "", "Memory per VM (e.g., 2Gi)")
	f.String("disk-size", "", "Data disk size")
	f.String("container-disk-image", "", "Container disk image for VMs")
	f.Bool("dry-run", false, "Print specs without creating resources")
	f.Bool("no-wait", false, "Skip waiting for VM readiness")
	f.Int("timeout", 0, "Readiness timeout in seconds")
	f.String("ssh-user", "", "SSH user for VMs")
	f.String("ssh-password", "", "SSH password for VMs")
	f.StringSlice("ssh-key", nil, "SSH authorized key (repeatable)")
	f.StringSlice("ssh-key-file", nil, "SSH key file path (repeatable)")

	return cmd
}

func newCleanupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Delete all managed resources",
		Long:  `Delete all VMs, services, and optionally the namespace created by virtwork.`,
		RunE:  cleanupE,
	}

	cmd.Flags().Bool("delete-namespace", false, "Also delete the namespace")
	return cmd
}

// vmPlan describes a single VM to be created during orchestration.
type vmPlan struct {
	workload  workloads.Workload
	vmSpec    *vm.VMSpecOpts
	vmName    string
	component string
}

// runE is the main orchestration flow for the "run" subcommand.
func runE(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig(cmd)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Determine which workloads to deploy
	workloadNames, _ := cmd.Flags().GetStringSlice("workloads")
	vmCountFlag, _ := cmd.Flags().GetInt("vm-count")

	registry := workloads.DefaultRegistry()
	registryOpts := []workloads.Option{
		workloads.WithNamespace(cfg.Namespace),
		workloads.WithSSHCredentials(cfg.SSHUser, cfg.SSHPassword, cfg.SSHAuthorizedKeys),
		workloads.WithDataDiskSize(cfg.DataDiskSize),
	}

	// Build workload instances
	var plans []vmPlan
	var vmNames []string

	for _, name := range workloadNames {
		wlCfg := config.WorkloadConfig{
			Enabled:  true,
			VMCount:  vmCountFlag,
			CPUCores: cfg.CPUCores,
			Memory:   cfg.Memory,
		}
		// Override with per-workload config from YAML if present
		if fileCfg, ok := cfg.Workloads[name]; ok {
			if fileCfg.CPUCores > 0 {
				wlCfg.CPUCores = fileCfg.CPUCores
			}
			if fileCfg.Memory != "" {
				wlCfg.Memory = fileCfg.Memory
			}
			if fileCfg.VMCount > 0 {
				wlCfg.VMCount = fileCfg.VMCount
			}
		}

		w, err := registry.Get(name, wlCfg, registryOpts...)
		if err != nil {
			return fmt.Errorf("creating workload %q: %w", name, err)
		}

		vmCount := w.VMCount()
		res := w.VMResources()

		if _, isMulti := w.(workloads.MultiVMWorkload); !isMulti {
			userdata, err := w.CloudInitUserdata()
			if err != nil {
				return fmt.Errorf("generating cloud-init for %q: %w", name, err)
			}

			for i := 0; i < vmCount; i++ {
				vmName := fmt.Sprintf("virtwork-%s-%d", name, i)
				plans = append(plans, vmPlan{
					workload:  w,
					component: name,
					vmName:    vmName,
					vmSpec: &vm.VMSpecOpts{
						Name:               vmName,
						Namespace:          cfg.Namespace,
						ContainerDiskImage: cfg.ContainerDiskImage,
						CloudInitUserdata:  userdata,
						CPUCores:           res.CPUCores,
						Memory:             res.Memory,
						Labels: map[string]string{
							constants.LabelAppName:   fmt.Sprintf("virtwork-%s", name),
							constants.LabelManagedBy: constants.ManagedByValue,
							constants.LabelComponent: name,
						},
						ExtraDisks:          w.ExtraDisks(),
						ExtraVolumes:        w.ExtraVolumes(),
						DataVolumeTemplates: w.DataVolumeTemplates(),
					},
				})
				vmNames = append(vmNames, vmName)
			}
		} else {
			// Multi-VM workload — use UserdataForRole
			multiVM, ok := w.(workloads.MultiVMWorkload)
			if !ok {
				return fmt.Errorf("workload %q reports VMCount=%d but does not implement MultiVMWorkload", name, vmCount)
			}

			roles := []string{"server", "client"}
			for i, role := range roles {
				userdata, err := multiVM.UserdataForRole(role, cfg.Namespace)
				if err != nil {
					return fmt.Errorf("generating cloud-init for %q role %q: %w", name, role, err)
				}

				vmName := fmt.Sprintf("virtwork-%s-%s-%d", name, role, i)
				labels := map[string]string{
					constants.LabelAppName:   fmt.Sprintf("virtwork-%s", name),
					constants.LabelManagedBy: constants.ManagedByValue,
					constants.LabelComponent: name,
					"virtwork/role":          role,
				}
				plans = append(plans, vmPlan{
					workload:  w,
					component: name,
					vmName:    vmName,
					vmSpec: &vm.VMSpecOpts{
						Name:               vmName,
						Namespace:          cfg.Namespace,
						ContainerDiskImage: cfg.ContainerDiskImage,
						CloudInitUserdata:  userdata,
						CPUCores:           res.CPUCores,
						Memory:             res.Memory,
						Labels:             labels,
						ExtraDisks:         w.ExtraDisks(),
						ExtraVolumes:       w.ExtraVolumes(),
					},
				})
				vmNames = append(vmNames, vmName)
			}
		}
	}

	// Dry-run: print specs and return
	if cfg.DryRun {
		return printDryRun(plans)
	}

	// Connect to cluster
	c, err := cluster.Connect(cfg.KubeconfigPath)
	if err != nil {
		return fmt.Errorf("connecting to cluster: %w", err)
	}

	ctx := context.Background()

	// Ensure namespace exists
	if err := resources.EnsureNamespace(ctx, c, cfg.Namespace, map[string]string{
		constants.LabelManagedBy: constants.ManagedByValue,
	}); err != nil {
		return fmt.Errorf("ensuring namespace %q: %w", cfg.Namespace, err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Namespace %s ensured\n", cfg.Namespace)

	// Create services before VMs (DNS must resolve for client VMs)
	servicesCreated := 0
	for _, name := range workloadNames {
		// Re-fetch workload to check service requirement
		wlCfg := config.WorkloadConfig{
			Enabled:  true,
			VMCount:  vmCountFlag,
			CPUCores: cfg.CPUCores,
			Memory:   cfg.Memory,
		}
		w, err := registry.Get(name, wlCfg, registryOpts...)
		if err != nil {
			continue
		}
		if w.RequiresService() {
			svc := w.ServiceSpec()
			if svc != nil {
				if err := resources.CreateService(ctx, c, svc); err != nil {
					return fmt.Errorf("creating service for %q: %w", name, err)
				}
				servicesCreated++
				fmt.Fprintf(cmd.OutOrStdout(), "Service %s created\n", svc.Name)
			}
		}
	}

	// Create VMs concurrently via errgroup
	g, gctx := errgroup.WithContext(ctx)
	for _, p := range plans {
		p := p // capture loop variable
		g.Go(func() error {
			vmObj := vm.BuildVMSpec(*p.vmSpec)
			if err := vm.CreateVM(gctx, c, vmObj); err != nil {
				return fmt.Errorf("creating VM %q: %w", p.vmName, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "VM %s created\n", p.vmName)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("creating VMs: %w", err)
	}

	// Wait for readiness
	if cfg.WaitForReady {
		timeout := time.Duration(cfg.ReadyTimeoutSeconds) * time.Second
		fmt.Fprintf(cmd.OutOrStdout(), "Waiting for %d VMs to become ready (timeout: %s)...\n",
			len(vmNames), timeout)
		results := wait.WaitForAllVMsReady(ctx, c, vmNames, cfg.Namespace,
			timeout, constants.DefaultPollInterval)

		failures := 0
		for name, err := range results {
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "VM %s: %v\n", name, err)
				failures++
			}
		}
		if failures > 0 {
			return fmt.Errorf("%d of %d VMs failed readiness check", failures, len(vmNames))
		}
		fmt.Fprintf(cmd.OutOrStdout(), "All %d VMs ready\n", len(vmNames))
	}

	// Print summary
	printSummary(cmd, len(plans), servicesCreated, cfg)
	return nil
}

// cleanupE is the cleanup flow for the "cleanup" subcommand.
func cleanupE(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig(cmd)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	deleteNS, _ := cmd.Flags().GetBool("delete-namespace")

	c, err := cluster.Connect(cfg.KubeconfigPath)
	if err != nil {
		return fmt.Errorf("connecting to cluster: %w", err)
	}

	ctx := context.Background()
	result, err := cleanup.CleanupAll(ctx, c, cfg.Namespace, deleteNS)
	if err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Cleanup complete: %d VMs deleted, %d services deleted",
		result.VMsDeleted, result.ServicesDeleted)
	if result.NamespaceDeleted {
		fmt.Fprintf(cmd.OutOrStdout(), ", namespace deleted")
	}
	fmt.Fprintln(cmd.OutOrStdout())

	if len(result.Errors) > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warnings (%d):\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Fprintf(cmd.ErrOrStderr(), "  - %v\n", e)
		}
	}

	return nil
}

// printDryRun outputs VM specs in YAML without connecting to a cluster.
func printDryRun(plans []vmPlan) error {
	fmt.Println("--- Dry Run ---")
	fmt.Printf("Total VMs to create: %d\n\n", len(plans))

	for _, p := range plans {
		vmObj := vm.BuildVMSpec(*p.vmSpec)
		data, err := sigyaml.Marshal(vmObj)
		if err != nil {
			return fmt.Errorf("marshaling VM spec for %q: %w", p.vmName, err)
		}
		fmt.Printf("# VM: %s (workload: %s)\n", p.vmName, p.component)
		fmt.Println(string(data))
		fmt.Println("---")
	}
	return nil
}

// printSummary outputs a deployment summary table.
func printSummary(cmd *cobra.Command, vmCount, svcCount int, cfg *config.Config) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, strings.Repeat("=", 50))
	fmt.Fprintln(out, "Deployment Summary")
	fmt.Fprintln(out, strings.Repeat("=", 50))
	fmt.Fprintf(out, "Namespace:    %s\n", cfg.Namespace)
	fmt.Fprintf(out, "VMs created:  %d\n", vmCount)
	fmt.Fprintf(out, "Services:     %d\n", svcCount)
	fmt.Fprintf(out, "Image:        %s\n", cfg.ContainerDiskImage)
	fmt.Fprintln(out, strings.Repeat("=", 50))
}
