package service

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
)

// LoginUser returns the interactive login account name for ownership checks.
func LoginUser(exec gitexec.Exec) (string, error) {
	if exec == nil {
		return "", fmt.Errorf("service: exec is nil")
	}
	if runtime.GOOS == "windows" {
		u := strings.TrimSpace(os.Getenv("USERNAME"))
		if u != "" {
			return u, nil
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.Run(ctx, "whoami")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// UsersMatch reports whether service logon matches the login user (Windows domain rules).
func UsersMatch(loginUser, serviceUser string) bool {
	login := normalizeAccount(loginUser)
	svc := normalizeAccount(serviceUser)
	if login == "" || svc == "" {
		return false
	}
	return strings.EqualFold(login, svc)
}

// IsLocalSystemLogon reports Windows LocalSystem-style accounts.
func IsLocalSystemLogon(serviceUser string) bool {
	u := strings.ToUpper(strings.TrimSpace(serviceUser))
	return u == "LOCALSYSTEM" || u == "NT AUTHORITY\\SYSTEM" || u == "SYSTEM"
}

func normalizeAccount(user string) string {
	u := strings.TrimSpace(user)
	u = strings.TrimPrefix(u, ".\\")
	if i := strings.LastIndex(u, "\\"); i >= 0 {
		u = u[i+1:]
	}
	return u
}
