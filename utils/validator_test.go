package utils

import (
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"email valide", "user@example.com", false},
		{"email valide avec sous-domaine", "user@mail.example.com", false},
		{"email vide", "", true},
		{"email sans @", "userexample.com", true},
		{"email sans domaine", "user@", true},
		{"email format invalide", "invalid", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"mot de passe valide", "password123", false},
		{"mot de passe court valide", "123456", false},
		{"mot de passe vide", "", true},
		{"mot de passe trop court", "12345", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name   string
		field  string
		value  string
		wantErr bool
	}{
		{"champ rempli", "name", "John", false},
		{"champ vide", "name", "", true},
		{"champ espaces uniquement", "name", "   ", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequired(tt.field, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequired() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
