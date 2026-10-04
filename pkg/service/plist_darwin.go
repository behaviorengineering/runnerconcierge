//go:build darwin

package service

import (
	"strings"
)

type launchdPlistInfo struct {
	Argv       []string
	ConfigPath string
	Schedule   string
}

func parseLaunchdPlist(data []byte) launchdPlistInfo {
	var info launchdPlistInfo
	s := string(data)
	argv := plistStringArrayAfterKey(s, "ProgramArguments")
	if len(argv) == 0 {
		if prog := plistStringAfterKey(s, "Program"); prog != "" {
			argv = []string{prog}
		}
	}
	info.Argv = argv
	info.ConfigPath = configPathFromArgv(argv)
	if strings.Contains(s, "<key>StartInterval</key>") {
		info.Schedule = "interval"
	} else if strings.Contains(s, "<key>StartCalendarInterval</key>") {
		info.Schedule = "calendar"
	}
	return info
}

func plistStringArrayAfterKey(plist, key string) []string {
	idx := strings.Index(plist, "<key>"+key+"</key>")
	if idx < 0 {
		return nil
	}
	rest := plist[idx:]
	arrStart := strings.Index(rest, "<array>")
	if arrStart < 0 {
		return nil
	}
	rest = rest[arrStart+len("<array>"):]
	arrEnd := strings.Index(rest, "</array>")
	if arrEnd < 0 {
		return nil
	}
	chunk := rest[:arrEnd]
	var out []string
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

func plistStringAfterKey(plist, key string) string {
	idx := strings.Index(plist, "<key>"+key+"</key>")
	if idx < 0 {
		return ""
	}
	rest := plist[idx+len("<key>"+key+"</key>"):]
	start := strings.Index(rest, "<string>")
	if start < 0 {
		return ""
	}
	rest = rest[start+len("<string>"):]
	end := strings.Index(rest, "</string>")
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func configPathFromArgv(argv []string) string {
	for i, a := range argv {
		if a == "--config" && i+1 < len(argv) {
			return strings.TrimSpace(argv[i+1])
		}
		if strings.HasPrefix(a, "--config=") {
			return strings.TrimSpace(strings.TrimPrefix(a, "--config="))
		}
	}
	return ""
}

func joinArgv(argv []string) string {
	return strings.Join(argv, " ")
}
