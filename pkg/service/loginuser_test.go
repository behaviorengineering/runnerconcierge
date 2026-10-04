package service

import "testing"

func TestUsersMatch(t *testing.T) {
	if !UsersMatch("hector", ".\\hector") {
		t.Fatal("expected domain-local match")
	}
	if UsersMatch("hector", "other") {
		t.Fatal("expected mismatch")
	}
}

func TestIsLocalSystemLogon(t *testing.T) {
	if !IsLocalSystemLogon("LocalSystem") {
		t.Fatal("expected LocalSystem")
	}
	if !IsLocalSystemLogon("NT AUTHORITY\\SYSTEM") {
		t.Fatal("expected SYSTEM")
	}
}
