package utils

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "testpassword123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "" {
		t.Error("HashPassword() returned empty hash")
	}
	if hash == password {
		t.Error("HashPassword() should not return plain password")
	}
}

func TestCheckPassword(t *testing.T) {
	password := "testpassword123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if !CheckPassword(hash, password) {
		t.Error("CheckPassword() should return true for correct password")
	}
	if CheckPassword(hash, "wrongpassword") {
		t.Error("CheckPassword() should return false for wrong password")
	}
	if CheckPassword(hash, "") {
		t.Error("CheckPassword() should return false for empty password")
	}
}
