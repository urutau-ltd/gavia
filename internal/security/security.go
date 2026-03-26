package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/mlkem"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha3"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	passwordIterations = 210_000
	passwordSaltSize   = 16
	tokenEntropyBytes  = 32
	sha3DigestSize     = 32
)

type EncryptedBackup struct {
	Format        string `json:"format"`
	Salt          string `json:"salt"`
	KEMCiphertext string `json:"kem_ciphertext"`
	Ciphertext    string `json:"ciphertext"`
}

func HashPassword(password string) (string, error) {
	password = strings.TrimSpace(password)
	if len(password) < 8 {
		return "", errors.New("password must be at least 8 characters long")
	}

	salt := make([]byte, passwordSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	digest, err := pbkdf2.Key(sha3.New256, password, salt, passwordIterations, sha3DigestSize)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"pbkdf2-sha3-256$%d$%s$%s",
		passwordIterations,
		encode(salt),
		encode(digest),
	), nil
}

func VerifyPassword(encodedHash, password string) bool {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha3-256" {
		return false
	}

	iter, err := strconv.Atoi(parts[1])
	if err != nil || iter <= 0 {
		return false
	}

	salt, err := decode(parts[2])
	if err != nil {
		return false
	}

	expected, err := decode(parts[3])
	if err != nil {
		return false
	}

	got, err := pbkdf2.Key(sha3.New256, password, salt, iter, len(expected))
	if err != nil {
		return false
	}

	return subtle.ConstantTimeCompare(expected, got) == 1
}

func GenerateToken() (plain, hashed, hint string, err error) {
	plain, err = randomToken()
	if err != nil {
		return "", "", "", err
	}

	hashed = HashToken(plain)
	hint = plain
	if len(plain) > 8 {
		hint = plain[len(plain)-8:]
	}

	return plain, hashed, hint, nil
}

func HashToken(token string) string {
	sum := sha3.Sum256([]byte(token))
	return encode(sum[:])
}

func GenerateRecoveryKeyPair() (publicKey, seed string, err error) {
	dk, err := mlkem.GenerateKey768()
	if err != nil {
		return "", "", err
	}

	return encode(dk.EncapsulationKey().Bytes()), encode(dk.Bytes()), nil
}

func RecoverySeedMatchesPublicKey(seed, publicKey string) bool {
	seedBytes, err := decode(seed)
	if err != nil {
		return false
	}

	publicKeyBytes, err := decode(publicKey)
	if err != nil {
		return false
	}

	dk, err := mlkem.NewDecapsulationKey768(seedBytes)
	if err != nil {
		return false
	}

	return subtle.ConstantTimeCompare(dk.EncapsulationKey().Bytes(), publicKeyBytes) == 1
}

func EncryptBackup(plaintext []byte, publicKey string) (*EncryptedBackup, error) {
	publicKeyBytes, err := decode(publicKey)
	if err != nil {
		return nil, err
	}

	ek, err := mlkem.NewEncapsulationKey768(publicKeyBytes)
	if err != nil {
		return nil, err
	}

	sharedKey, kemCiphertext := ek.Encapsulate()
	return sealBackup(plaintext, sharedKey, kemCiphertext)
}

func DecryptBackup(bundle EncryptedBackup, recoverySeed string) ([]byte, error) {
	seedBytes, err := decode(recoverySeed)
	if err != nil {
		return nil, err
	}

	kemCiphertext, err := decode(bundle.KEMCiphertext)
	if err != nil {
		return nil, err
	}

	dk, err := mlkem.NewDecapsulationKey768(seedBytes)
	if err != nil {
		return nil, err
	}

	sharedKey, err := dk.Decapsulate(kemCiphertext)
	if err != nil {
		return nil, err
	}

	return openBackup(bundle, sharedKey)
}

func sealBackup(plaintext, sharedKey, kemCiphertext []byte) (*EncryptedBackup, error) {
	salt := make([]byte, sha3DigestSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	key, err := hkdf.Key(sha3.New256, sharedKey, salt, "gavia-backup", 32)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return &EncryptedBackup{
		Format:        "gavia.backup.mlkem768+aes256gcm",
		Salt:          encode(salt),
		KEMCiphertext: encode(kemCiphertext),
		Ciphertext:    encode(append(nonce, ciphertext...)),
	}, nil
}

func openBackup(bundle EncryptedBackup, sharedKey []byte) ([]byte, error) {
	if strings.TrimSpace(bundle.Format) != "gavia.backup.mlkem768+aes256gcm" {
		return nil, errors.New("unsupported encrypted backup format")
	}

	salt, err := decode(bundle.Salt)
	if err != nil {
		return nil, err
	}

	key, err := hkdf.Key(sha3.New256, sharedKey, salt, "gavia-backup", 32)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	ciphertext, err := decode(bundle.Ciphertext)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("encrypted backup payload is too short")
	}

	nonce, encryptedPayload := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, encryptedPayload, nil)
}

func randomToken() (string, error) {
	random := make([]byte, tokenEntropyBytes)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}

	return encode(random), nil
}

func encode(value []byte) string {
	return base64.RawURLEncoding.EncodeToString(value)
}

func decode(value string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(strings.TrimSpace(value))
}
