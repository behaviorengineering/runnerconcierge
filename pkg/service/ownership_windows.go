//go:build windows

package service

import (
	"encoding/json"
	"strings"
)

type winSvcJSON struct {
	Name      string `json:"Name"`
	State     string `json:"State"`
	StartName string `json:"StartName"`
	PathName  string `json:"PathName"`
}

func parseWindowsOwnershipJSON(raw string) ([]Ownership, error) {
	var list []winSvcJSON
	if strings.HasPrefix(raw, "[") {
		if err := json.Unmarshal([]byte(raw), &list); err != nil {
			return nil, err
		}
	} else {
		var one winSvcJSON
		if err := json.Unmarshal([]byte(raw), &one); err != nil {
			return nil, err
		}
		list = []winSvcJSON{one}
	}
	out := make([]Ownership, 0, len(list))
	for _, s := range list {
		cmd := SanitizeCommand(s.PathName)
		state := s.State
		processUp := strings.EqualFold(state, "running")
		out = append(out, Ownership{
			ServiceName: s.Name,
			State:       state,
			LogonUser:   s.StartName,
			ConfigPath:  parseConfigFromPathName(s.PathName),
			Kind:        "windows_service",
			Command:     cmd,
			MatchReason: MatchReasonWindowsServiceName,
			Role:        ClassifyRole(s.Name, "windows_service", cmd),
			ProcessUp:   processUp,
		})
	}
	return out, nil
}

func parseConfigFromPathName(pathName string) string {
	lower := strings.ToLower(pathName)
	idx := strings.Index(lower, "--config")
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(pathName[idx+len("--config"):])
	rest = strings.TrimPrefix(rest, "=")
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return ""
	}
	if strings.HasPrefix(rest, "\"") {
		rest = strings.TrimPrefix(rest, "\"")
		if j := strings.Index(rest, "\""); j >= 0 {
			return rest[:j]
		}
	}
	fields := strings.Fields(rest)
	if len(fields) > 0 {
		return strings.Trim(fields[0], "\"")
	}
	return ""
}
