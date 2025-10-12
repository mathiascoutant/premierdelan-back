package utils

import (
	"fmt"
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
var phoneRegex = regexp.MustCompile(`^(\+33|0)[1-9](\d{2}){4}$`)

// ValidationError représente une erreur de validation
type ValidationError struct {
	Field   string
	Message string
}

// Error implémente l'interface error
func (v ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", v.Field, v.Message)
}

// ValidateEmail valide un email
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return ValidationError{Field: "email", Message: "l'email est requis"}
	}
	if !emailRegex.MatchString(email) {
		return ValidationError{Field: "email", Message: "format d'email invalide"}
	}
	return nil
}

// ValidatePassword valide un mot de passe
func ValidatePassword(password string) error {
	if password == "" {
		return ValidationError{Field: "password", Message: "le mot de passe est requis"}
	}
	if len(password) < 6 {
		return ValidationError{Field: "password", Message: "le mot de passe doit contenir au moins 6 caractères"}
	}
	return nil
}

// ValidateRequired valide qu'un champ n'est pas vide
func ValidateRequired(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return ValidationError{Field: field, Message: fmt.Sprintf("le champ %s est requis", field)}
	}
	return nil
}

// ValidatePhone valide un numéro de téléphone français
func ValidatePhone(phone string) error {
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, ".", "")
	phone = strings.ReplaceAll(phone, "-", "")
	
	if phone == "" {
		return ValidationError{Field: "telephone", Message: "le numéro de téléphone est requis"}
	}
	
	// Accepter les formats français courants
	if !phoneRegex.MatchString(phone) && len(phone) < 10 {
		return ValidationError{Field: "telephone", Message: "format de téléphone invalide"}
	}
	
	return nil
}

