package security

import "testing"

func TestHashPasswordAndVerify(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if !VerifyPassword(hash, "correct horse battery staple") {
		t.Fatal("expected password verification to succeed")
	}

	if VerifyPassword(hash, "wrong password") {
		t.Fatal("expected password verification to fail for the wrong password")
	}

	if !VerifyPassword(hash, "  correct horse battery staple  ") {
		t.Fatal("expected password verification to tolerate surrounding whitespace the same way hashing does")
	}
}

func TestRecoveryKeyAndEncryptedBackupRoundTrip(t *testing.T) {
	publicKey, recoveryKey, err := GenerateRecoveryKeyPair()
	if err != nil {
		t.Fatalf("GenerateRecoveryKeyPair returned error: %v", err)
	}

	if !RecoverySeedMatchesPublicKey(recoveryKey, publicKey) {
		t.Fatal("expected recovery key to match the generated public key")
	}

	if !RecoverySeedMatchesPublicKey("\n"+recoveryKey+"\n", "  "+publicKey+"  ") {
		t.Fatal("expected recovery key comparison to tolerate pasted whitespace")
	}

	bundle, err := EncryptBackup([]byte(`{"format":"gavia.backup.v1"}`), publicKey)
	if err != nil {
		t.Fatalf("EncryptBackup returned error: %v", err)
	}

	plaintext, err := DecryptBackup(*bundle, "\n"+recoveryKey+"\n")
	if err != nil {
		t.Fatalf("DecryptBackup returned error: %v", err)
	}

	if got := string(plaintext); got != `{"format":"gavia.backup.v1"}` {
		t.Fatalf("expected decrypted payload to round-trip, got %q", got)
	}
}
