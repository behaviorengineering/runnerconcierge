//go:build darwin

package cleanup

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

func platformInstallSchedule(ctx context.Context, exec gitexec.Exec, self string, interval time.Duration, allowYes bool, out io.Writer) error {
	legacy, err := findLegacyScriptAgents()
	if err != nil {
		return err
	}
	if len(legacy) > 0 && !allowYes {
		return errdefs.New("cleanup.Install", errdefs.CodeProcessConflict,
			fmt.Sprintf("found legacy cleanup agent(s) %v; re-run with --yes to replace", legacy), nil)
	}
	for _, label := range legacy {
		if err := unloadLaunchAgent(ctx, exec, label); err != nil {
			return err
		}
		plist := filepath.Join(launchAgentsDir(), label+".plist")
		if err := os.Remove(plist); err != nil && !os.IsNotExist(err) {
			return err
		}
		if out != nil {
			fmt.Fprintf(out, "Removed legacy cleanup agent %s\n", label)
		}
	}

	plistPath := filepath.Join(launchAgentsDir(), UnitName+".plist")
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return err
	}
	logDir := filepath.Join(userHome(), "Library", "Logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return err
	}
	intervalSec := int(interval.Seconds())
	if intervalSec < 60 {
		intervalSec = 60
	}
	content := darwinPlist(self, intervalSec, logDir)
	if err := os.WriteFile(plistPath, []byte(content), 0o644); err != nil {
		return err
	}
	if err := unloadLaunchAgent(ctx, exec, UnitName); err != nil && !launchctlBenign(err) {
		return err
	}
	domain := launchdDomain()
	if _, err := exec.Run(ctx, "launchctl", "bootstrap", domain, plistPath); err != nil {
		return errdefs.New("cleanup.Install", errdefs.CodeServiceStart, "launchctl bootstrap", err)
	}
	if out != nil {
		fmt.Fprintf(out, "Installed cleanup agent %s (interval %ds)\n", UnitName, intervalSec)
	}
	return nil
}

func platformUninstallSchedule(ctx context.Context, exec gitexec.Exec, out io.Writer) error {
	if err := unloadLaunchAgent(ctx, exec, UnitName); err != nil && !launchctlBenign(err) {
		return err
	}
	plistPath := filepath.Join(launchAgentsDir(), UnitName+".plist")
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if out != nil {
		fmt.Fprintf(out, "Uninstalled cleanup agent %s\n", UnitName)
	}
	return nil
}

func darwinPlist(self string, intervalSec int, logDir string) string {
	stdout := filepath.Join(logDir, UnitName+".stdout.log")
	stderr := filepath.Join(logDir, UnitName+".stderr.log")
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>cleanup</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>StartInterval</key>
	<integer>%d</integer>
	<key>StandardOutPath</key>
	<string>%s</string>
	<key>StandardErrorPath</key>
	<string>%s</string>
</dict>
</plist>
`, UnitName, self, intervalSec, stdout, stderr)
}

func launchAgentsDir() string {
	return filepath.Join(userHome(), "Library", "LaunchAgents")
}

func userHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func launchdDomain() string {
	uid := os.Getuid()
	return "gui/" + strconv.Itoa(uid)
}

func unloadLaunchAgent(ctx context.Context, exec gitexec.Exec, label string) error {
	target := launchdDomain() + "/" + label
	_, err := exec.Run(ctx, "launchctl", "bootout", target)
	return err
}

func launchctlBenign(err error) bool {
	if err == nil {
		return true
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "no such process") || strings.Contains(s, "could not find") || strings.Contains(s, "not found")
}

func findLegacyScriptAgents() ([]string, error) {
	dir := launchAgentsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var labels []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".plist") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		if plistReferencesBasename(data, LegacyScriptBasename) {
			labels = append(labels, strings.TrimSuffix(e.Name(), ".plist"))
		}
	}
	return labels, nil
}

func plistReferencesBasename(data []byte, basename string) bool {
	s := string(data)
	for _, part := range plistStringValues(s) {
		if filepath.Base(strings.TrimSpace(part)) == basename {
			return true
		}
	}
	return false
}

func plistStringValues(plist string) []string {
	var out []string
	chunk := plist
	for {
		start := strings.Index(chunk, "<string>")
		if start < 0 {
			break
		}
		chunk = chunk[start+len("<string>"):]
		end := strings.Index(chunk, "</string>")
		if end < 0 {
			break
		}
		out = append(out, chunk[:end])
		chunk = chunk[end+len("</string>"):]
	}
	return out
}
