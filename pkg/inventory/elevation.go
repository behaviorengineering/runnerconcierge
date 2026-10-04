package inventory

import (
	"os/exec"
	"runtime"
	"strings"
)

func isElevated() bool {
	if runtime.GOOS != "windows" {
		return false
	}
	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		"([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "True"
}
