package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/behaviorengineering/runnerconcierge/internal/config"
	"github.com/behaviorengineering/runnerconcierge/pkg/preset"
	"github.com/behaviorengineering/runnerconcierge/pkg/wizard"
	"github.com/spf13/cobra"
)

func dispatchCobra(ctx context.Context, args []string, w io.Writer) (int, bool) {
	if len(args) == 0 {
		return 0, false
	}
	root := &cobra.Command{
		Use:           "runnerconcierge",
		Short:         "GitLab self-hosted runner setup wizard",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(w)
	root.SetErr(w)

	var initForce bool
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Seed user config.yaml",
		RunE: func(_ *cobra.Command, _ []string) error {
			path, err := config.Init(initForce)
			if err != nil {
				return err
			}
			fmt.Fprintf(w, "config: %s\n", path)
			return nil
		},
	}
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite config.yaml")

	var doctorDocker bool
	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "Preflight report",
		RunE: func(_ *cobra.Command, _ []string) error {
			code := runDoctor(ctx, []string{"-docker=" + fmt.Sprintf("%t", doctorDocker)}, w)
			if code == ExitDoctor {
				return fmt.Errorf("doctor found blocking issues")
			}
			if code != ExitOK {
				return fmt.Errorf("doctor failed")
			}
			return nil
		},
	}
	doctorCmd.Flags().BoolVar(&doctorDocker, "docker", false, "require docker")

	verifyCmd := &cobra.Command{
		Use:   "verify",
		Short: "Service status",
		RunE: func(_ *cobra.Command, _ []string) error {
			if runVerify(ctx, nil, w) != ExitOK {
				return fmt.Errorf("verify failed")
			}
			return nil
		},
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Runner inventory and smells",
		RunE: func(cmd *cobra.Command, _ []string) error {
			code := runStatus(ctx, statusArgs(cmd), w)
			if code == ExitDoctor {
				return fmt.Errorf("status found blocking issues")
			}
			if code != ExitOK {
				return fmt.Errorf("status failed")
			}
			return nil
		},
	}
	statusCmd.Flags().Bool("json", false, "JSON output")
	statusCmd.Flags().StringArray("runner-config", nil, "extra runner config.toml paths")

	repairCmd := &cobra.Command{
		Use:   "repair-service",
		Short: "Rebind runner service to login user",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if runRepairService(ctx, repairArgs(cmd), w) != ExitOK {
				return fmt.Errorf("repair-service failed")
			}
			return nil
		},
	}
	repairCmd.Flags().String("runner-config", "", "runner config.toml path")
	repairCmd.Flags().String("service", "", "service name when multiple")
	repairCmd.Flags().Bool("yes", false, "allow destructive repair")
	repairCmd.Flags().String("windows-password", "", "Windows service password")
	repairCmd.Flags().Bool("brew", false, "use brew services on macOS")

	setupFlags := &wizard.Options{Preset: preset.Preset{}}
	var setupTagList string
	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Run setup wizard with flags",
		RunE: func(_ *cobra.Command, _ []string) error {
			setupFlags.TagList = splitTags(setupTagList)
			if runWizard(ctx, w, setupFlags.Preset, setupFlags, false) != ExitOK {
				return fmt.Errorf("setup failed")
			}
			return nil
		},
	}
	bindWizardFlags(setupCmd, setupFlags, &setupTagList)

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print release identity",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Fprintf(w, "runnerconcierge %s\n", version)
		},
	}

	helpCmd := &cobra.Command{
		Use:   "help",
		Short: "Command catalog",
		Run: func(_ *cobra.Command, _ []string) {
			printHelp(w)
		},
	}

	root.AddCommand(initCmd, doctorCmd, verifyCmd, statusCmd, repairCmd, setupCmd, versionCmd, helpCmd)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		if strings.Contains(err.Error(), "unknown command") {
			printHelp(w)
			return ExitUsage, true
		}
		fmt.Fprintf(w, "error: %v\n", err)
		return ExitFail, true
	}
	return ExitOK, true
}

func bindWizardFlags(cmd *cobra.Command, o *wizard.Options, tagList *string) {
	cmd.Flags().BoolVar(&o.NonInteractive, "non-interactive", false, "no prompts")
	cmd.Flags().BoolVar(&o.Resume, "resume", false, "resume checkpoint")
	cmd.Flags().BoolVar(&o.Fresh, "fresh", false, "discard checkpoint")
	cmd.Flags().BoolVar(&o.AllowInstall, "yes", false, "allow installs")
	cmd.Flags().StringVar(&o.RunnerToken, "token", "", "glrt runner token")
	cmd.Flags().StringVar(&o.PAT, "pat", "", "gitlab PAT with create_runner")
	cmd.Flags().StringVar(&o.ProjectPath, "repo", "", "group/project path")
	cmd.Flags().StringVar(&o.Executor, "executor", "", "shell or docker")
	cmd.Flags().StringVar(tagList, "tag-list", "", "comma-separated tags")
	cmd.Flags().StringVar(&o.ConfigPath, "config", "", "config.yaml path")
}
