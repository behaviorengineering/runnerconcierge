package errdefs

import "fmt"

// Code classifies domain failures for operators and agents.
type Code string

const (
	CodeUnsupportedOS         Code = "unsupported_os"
	CodeGitMissing            Code = "git_missing"
	CodeProcessConflict       Code = "process_conflict"
	CodeInstallRefused        Code = "install_refused"
	CodeDownloadFailed        Code = "download_failed"
	CodeAuthRequired          Code = "auth_required"
	CodeAuthUnauthorized      Code = "auth_unauthorized"
	CodeAuthScopeInsufficient Code = "auth_scope_insufficient"
	CodeInvalidScope          Code = "invalid_scope"
	CodeCreateFailed          Code = "create_failed"
	CodeRegisterFailed        Code = "register_failed"
	CodeElevationRequired     Code = "elevation_required"
	CodeServiceLogon          Code = "service_logon"
	CodeServiceStart          Code = "service_start"
	CodeRunnerOffline         Code = "runner_offline"
	CodeMissingDeadline       Code = "missing_deadline"
	CodeBadServiceLogon       Code = "bad_service_logon"
	CodeWrongServiceUser      Code = "wrong_service_user"
	CodeSystemConfig          Code = "system_config"
	CodeConfigUnreadable      Code = "config_unreadable"
	CodeServiceMissing        Code = "service_missing"
	CodeServiceStopped        Code = "service_stopped"
	CodeRunnerBinaryMissing   Code = "runner_binary_missing"
	CodeUnsupportedForge      Code = "unsupported_forge"
	CodeDockerUnavailable     Code = "docker_unavailable"
)

// Error is a typed domain error with stable code.
type Error struct {
	Op   string
	Code Code
	Msg  string
	Err  error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Code != "" {
		return fmt.Sprintf("%s: %s (code=%s)", e.Op, e.Msg, e.Code)
	}
	return fmt.Sprintf("%s: %s", e.Op, e.Msg)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// New builds a domain error.
func New(op string, code Code, msg string, err error) *Error {
	return &Error{Op: op, Code: code, Msg: msg, Err: err}
}
