package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func statusArgs(cmd *cobra.Command) []string {
	var args []string
	if cmd.Flags().Changed("json") {
		if v, _ := cmd.Flags().GetBool("json"); v {
			args = append(args, "--json")
		}
	}
	if paths, err := cmd.Flags().GetStringArray("runner-config"); err == nil {
		for _, p := range paths {
			if strings.TrimSpace(p) != "" {
				args = append(args, "--runner-config="+strings.TrimSpace(p))
			}
		}
	}
	return args
}

func repairArgs(cmd *cobra.Command) []string {
	var args []string
	appendFlag := func(name string) {
		if !cmd.Flags().Changed(name) {
			return
		}
		v, err := cmd.Flags().GetString(name)
		if err == nil && strings.TrimSpace(v) != "" {
			args = append(args, fmt.Sprintf("--%s=%s", name, strings.TrimSpace(v)))
		}
	}
	appendFlag("runner-config")
	appendFlag("service")
	appendFlag("windows-password")
	if cmd.Flags().Changed("yes") {
		if v, _ := cmd.Flags().GetBool("yes"); v {
			args = append(args, "--yes")
		}
	}
	if cmd.Flags().Changed("brew") {
		if v, _ := cmd.Flags().GetBool("brew"); v {
			args = append(args, "--brew")
		}
	}
	return args
}
