//go:build !windows

package service

func parseWindowsOwnershipJSON(raw string) ([]Ownership, error) {
	return nil, nil
}
