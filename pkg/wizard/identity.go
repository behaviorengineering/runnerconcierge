package wizard

import (
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

// RunnerIdentity builds the stable runner name: parent path, tag, and hostname.
func RunnerIdentity(parent, tag, host string) (string, error) {
	parent = sanitizeIdentityPart(parent)
	tag = sanitizeIdentityPart(tag)
	host = sanitizeIdentityPart(host)
	if parent == "" || tag == "" {
		return "", errdefs.New("RunnerIdentity", errdefs.CodeInvalidScope,
			"runner parent path and tag are required", nil)
	}
	if host == "" {
		return "", errdefs.New("RunnerIdentity", errdefs.CodeInvalidScope, "hostname is required", nil)
	}
	return parent + "-" + tag + "-" + host, nil
}

func sanitizeIdentityPart(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, " ", "-")
	return s
}
