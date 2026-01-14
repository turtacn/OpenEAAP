// Package utils provides cryptographic utility functions for OpenEAAP.
// It includes AES encryption/decryption, hashing, signature verification,
// and field-level encryption support for privacy-preserving operations.
package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================================
// AES Encryption/Decryption Functions
// ============================================================================

// AESEncrypt encrypts plaintext using AES-256-GCM
// Returns base64-encoded ciphertext with nonce prepended
func AESEncrypt(plaintext []byte, key []byte) (string, error) {
	// Validate key length (must be 16, 24, or 32 bytes)
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return "", fmt.Errorf("invalid key length: must be 16, 24, or 32 bytes")
	}

	// Create AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher block: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// Return base64-encoded result
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// AESDecrypt decrypts base64-encoded ciphertext using AES-256-GCM
func AESDecrypt(ciphertext string, key []byte) ([]byte, error) {
	// Validate key length
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, fmt.Errorf("invalid key length: must be 16, 24, or 32 bytes")
	}

	// Decode base64 ciphertext
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	// Create AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce from ciphertext
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	// Decrypt and verify
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// AESEncryptString encrypts a string using AES-256-GCM
func AESEncryptString(plaintext string, key []byte) (string, error) {
	return AESEncrypt([]byte(plaintext), key)
}

// AESDecryptString decrypts a string using AES-256-GCM
func AESDecryptString(ciphertext string, key []byte) (string, error) {
	plaintext, err := AESDecrypt(ciphertext, key)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// ============================================================================
// Hash Functions
// ============================================================================

// SHA256Hash computes SHA-256 hash of data
func SHA256Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// SHA256HashString computes SHA-256 hash of string
func SHA256HashString(s string) string {
	return SHA256Hash([]byte(s))
}

// SHA512Hash computes SHA-512 hash of data
func SHA512Hash(data []byte) string {
	hash := sha512.Sum512(data)
	return hex.EncodeToString(hash[:])
}

// SHA512HashString computes SHA-512 hash of string
func SHA512HashString(s string) string {
	return SHA512Hash([]byte(s))
}

// HMACSHA256 computes HMAC-SHA256 of data with key
func HMACSHA256(data []byte, key []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// HMACSHA256String computes HMAC-SHA256 of string with key
func HMACSHA256String(s string, key []byte) string {
	return HMACSHA256([]byte(s), key)
}

// VerifyHMACSHA256 verifies HMAC-SHA256 signature
func VerifyHMACSHA256(data []byte, key []byte, signature string) bool {
	expected := HMACSHA256(data, key)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// ============================================================================
// Password Hashing Functions
// ============================================================================

// HashPassword hashes a password using bcrypt with default cost
func HashPassword(password string) (string, error) {
	return HashPasswordWithCost(password, bcrypt.DefaultCost)
}

// HashPasswordWithCost hashes a password using bcrypt with specified cost
func HashPasswordWithCost(password string, cost int) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword verifies a password against bcrypt hash
func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Argon2IDHash hashes data using Argon2id (recommended for passwords)
func Argon2IDHash(password string, salt []byte) string {
	// Argon2id parameters (recommended values)
	const (
		time    = 1
		memory  = 64 * 1024 // 64 MB
		threads = 4
		keyLen  = 32
	)

	hash := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)
	return base64.StdEncoding.EncodeToString(hash)
}

// Argon2IDHashWithSalt generates salt and hashes password
// Returns: base64(salt):base64(hash)
func Argon2IDHashWithSalt(password string) (string, error) {
	// Generate random salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := Argon2IDHash(password, salt)
	saltEncoded := base64.StdEncoding.EncodeToString(salt)

	return fmt.Sprintf("%s:%s", saltEncoded, hash), nil
}

// VerifyArgon2IDHash verifies password against Argon2id hash
// Expected format: base64(salt):base64(hash)
func VerifyArgon2IDHash(password, encoded string) bool {
	parts := splitN(encoded, ":", 2)
	if len(parts) != 2 {
		return false
	}

	salt, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}

	expectedHash := parts[1]
	actualHash := Argon2IDHash(password, salt)

	return hmac.Equal([]byte(expectedHash), []byte(actualHash))
}

// ============================================================================
// Random Generation Functions
// ============================================================================

// GenerateRandomBytes generates cryptographically secure random bytes
func GenerateRandomBytes(n int) ([]byte, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return bytes, nil
}

// GenerateRandomKey generates a random AES key of specified length
// length must be 16, 24, or 32 bytes (AES-128, AES-192, or AES-256)
func GenerateRandomKey(length int) ([]byte, error) {
	if length != 16 && length != 24 && length != 32 {
		return nil, fmt.Errorf("invalid key length: must be 16, 24, or 32 bytes")
	}
	return GenerateRandomBytes(length)
}

// GenerateAES256Key generates a random 256-bit AES key
func GenerateAES256Key() ([]byte, error) {
	return GenerateRandomKey(32)
}

// GenerateRandomToken generates a random token of specified byte length
// Returns base64-encoded token
func GenerateRandomToken(byteLength int) (string, error) {
	bytes, err := GenerateRandomBytes(byteLength)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// ============================================================================
// Field-Level Encryption Support
// ============================================================================

// FieldEncryptor provides field-level encryption capabilities
type FieldEncryptor struct {
	key []byte
}

// NewFieldEncryptor creates a new field encryptor with given key
func NewFieldEncryptor(key []byte) (*FieldEncryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes for AES-256")
	}
	return &FieldEncryptor{key: key}, nil
}

// Encrypt encrypts a field value
func (fe *FieldEncryptor) Encrypt(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	return AESEncryptString(value, fe.key)
}

// Decrypt decrypts a field value
func (fe *FieldEncryptor) Decrypt(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}
	return AESDecryptString(encrypted, fe.key)
}

// EncryptMap encrypts specified fields in a map
func (fe *FieldEncryptor) EncryptMap(data map[string]interface{}, fields []string) error {
	for _, field := range fields {
		if val, ok := data[field]; ok {
			if strVal, ok := val.(string); ok {
				encrypted, err := fe.Encrypt(strVal)
				if err != nil {
					return fmt.Errorf("failed to encrypt field %s: %w", field, err)
				}
				data[field] = encrypted
			}
		}
	}
	return nil
}

// DecryptMap decrypts specified fields in a map
func (fe *FieldEncryptor) DecryptMap(data map[string]interface{}, fields []string) error {
	for _, field := range fields {
		if val, ok := data[field]; ok {
			if strVal, ok := val.(string); ok {
				decrypted, err := fe.Decrypt(strVal)
				if err != nil {
					return fmt.Errorf("failed to decrypt field %s: %w", field, err)
				}
				data[field] = decrypted
			}
		}
	}
	return nil
}

// ============================================================================
// Data Integrity Functions
// ============================================================================

// ComputeChecksum computes SHA-256 checksum for data integrity
func ComputeChecksum(data []byte) string {
	return SHA256Hash(data)
}

// VerifyChecksum verifies data integrity using SHA-256
func VerifyChecksum(data []byte, expectedChecksum string) bool {
	actualChecksum := ComputeChecksum(data)
	return actualChecksum == expectedChecksum
}

// SignData signs data using HMAC-SHA256
func SignData(data []byte, secret []byte) string {
	return HMACSHA256(data, secret)
}

// VerifySignature verifies HMAC-SHA256 signature
func VerifySignature(data []byte, secret []byte, signature string) bool {
	return VerifyHMACSHA256(data, secret, signature)
}

// ============================================================================
// Helper Functions
// ============================================================================

// splitN splits string with limit (backport for older Go versions)
func splitN(s, sep string, n int) []string {
	parts := make([]string, 0, n)
	for i := 0; i < n-1; i++ {
		idx := indexOf(s, sep)
		if idx == -1 {
			break
		}
		parts = append(parts, s[:idx])
		s = s[idx+len(sep):]
	}
	if s != "" {
		parts = append(parts, s)
	}
	return parts
}

// indexOf finds first occurrence of substring
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// DeriveKey derives an encryption key from password using Argon2
func DeriveKey(password string, salt []byte, keyLen int) []byte {
	const (
		time    = 1
		memory  = 64 * 1024
		threads = 4
	)
	return argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(keyLen))
}

// DeriveKeyFromPassword derives a 256-bit key from password with random salt
// Returns: base64(salt):base64(key)
func DeriveKeyFromPassword(password string) (string, error) {
	salt, err := GenerateRandomBytes(16)
	if err != nil {
		return "", err
	}

	key := DeriveKey(password, salt, 32)

	saltEncoded := base64.StdEncoding.EncodeToString(salt)
	keyEncoded := base64.StdEncoding.EncodeToString(key)

	return fmt.Sprintf("%s:%s", saltEncoded, keyEncoded), nil
}

//Personal.AI order the ending
