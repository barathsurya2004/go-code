package utils

import (
	"testing"
)

func TestCreatePasswordAndCompare(t *testing.T) {
	password := "SecretPass123!"

	hashedPassword, err := CreatePassword(password)
	if err != nil {
		t.Fatalf("expected no error creating password, got %v", err)
	}

	ok, err := ComparePasswords(password, hashedPassword)
	if !ok || err != nil {
		t.Fatalf("expected ComparePasswords to succeed, got ok=%v, err=%v", ok, err)
	}

	okFail, errFail := ComparePasswords("WrongPass", hashedPassword)
	if okFail || errFail == nil {
		t.Fatalf("expected ComparePasswords to fail for wrong password, got ok=%v, err=%v", okFail, errFail)
	}
}
