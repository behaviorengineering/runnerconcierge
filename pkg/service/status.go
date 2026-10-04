package service

import (
	"strings"
)

const gitlabServiceNotInstalledPhrase = "the service is not installed"

func gitlabServiceNotInstalled(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), gitlabServiceNotInstalledPhrase)
}

func nativeStatusLine(out []byte) string {
	var last string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(stripANSI(line))
		if line == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "runtime platform") {
			continue
		}
		last = line
	}
	if last == "" {
		return "gitlab-runner service is running"
	}
	return last
}

func brewGitLabRunnerStatusLine(listOutput string) string {
	for _, line := range strings.Split(listOutput, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[0] != "gitlab-runner" {
			continue
		}
		state := fields[1]
		user := ""
		if len(fields) >= 3 && fields[2] != "Name" {
			user = fields[2]
		}
		if user != "" {
			return "gitlab-runner brew_services " + state + " user=" + user
		}
		return "gitlab-runner brew_services " + state
	}
	return ""
}

func stripANSI(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) {
				c := s[i]
				if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
					break
				}
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
