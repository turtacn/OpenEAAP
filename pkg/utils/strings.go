// Package utils provides common utility functions for string manipulation.
// It includes case conversion, masking, truncation, and random string generation
// to be reused across all layers of the OpenEAAP platform.
package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// ============================================================================
// Case Conversion Functions
// ============================================================================

// ToCamelCase converts a string to camelCase
// Example: "hello_world" -> "helloWorld", "Hello-World" -> "helloWorld"
func ToCamelCase(s string) string {
	if s == "" {
		return ""
	}

	// Split by common delimiters
	words := splitWords(s)
	if len(words) == 0 {
		return ""
	}

	// First word lowercase, rest title case
	result := strings.ToLower(words[0])
	for i := 1; i < len(words); i++ {
		if words[i] != "" {
			result += strings.Title(strings.ToLower(words[i]))
		}
	}

	return result
}

// ToPascalCase converts a string to PascalCase
// Example: "hello_world" -> "HelloWorld", "hello-world" -> "HelloWorld"
func ToPascalCase(s string) string {
	if s == "" {
		return ""
	}

	words := splitWords(s)
	var result strings.Builder

	for _, word := range words {
		if word != "" {
			result.WriteString(strings.Title(strings.ToLower(word)))
		}
	}

	return result.String()
}

// ToSnakeCase converts a string to snake_case
// Example: "HelloWorld" -> "hello_world", "helloWorld" -> "hello_world"
func ToSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder

	for i, r := range s {
		if unicode.IsUpper(r) {
			// Add underscore before uppercase letter (except at start)
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else if r == '-' || r == ' ' {
			// Replace delimiter with underscore
			result.WriteRune('_')
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// ToKebabCase converts a string to kebab-case
// Example: "HelloWorld" -> "hello-world", "hello_world" -> "hello-world"
func ToKebabCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder

	for i, r := range s {
		if unicode.IsUpper(r) {
			// Add hyphen before uppercase letter (except at start)
			if i > 0 {
				result.WriteRune('-')
			}
			result.WriteRune(unicode.ToLower(r))
		} else if r == '_' || r == ' ' {
			// Replace delimiter with hyphen
			result.WriteRune('-')
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// splitWords splits a string into words by common delimiters
func splitWords(s string) []string {
	// Split by underscore, hyphen, space, or camelCase
	re := regexp.MustCompile(`[_\-\s]+|(?<=[a-z])(?=[A-Z])|(?<=[A-Z])(?=[A-Z][a-z])`)
	return re.Split(s, -1)
}

// ============================================================================
// String Manipulation Functions
// ============================================================================

// Truncate truncates a string to the specified length
// If the string is longer, it appends suffix (default "...")
func Truncate(s string, length int) string {
	return TruncateWithSuffix(s, length, "...")
}

// TruncateWithSuffix truncates a string with a custom suffix
func TruncateWithSuffix(s string, length int, suffix string) string {
	if length <= 0 {
		return ""
	}

	runes := []rune(s)
	if len(runes) <= length {
		return s
	}

	suffixLen := len([]rune(suffix))
	if length <= suffixLen {
		return string(runes[:length])
	}

	return string(runes[:length-suffixLen]) + suffix
}

// TruncateWords truncates a string to the specified number of words
func TruncateWords(s string, wordCount int, suffix string) string {
	if wordCount <= 0 {
		return ""
	}

	words := strings.Fields(s)
	if len(words) <= wordCount {
		return s
	}

	return strings.Join(words[:wordCount], " ") + suffix
}

// Reverse reverses a string
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// RemoveWhitespace removes all whitespace from a string
func RemoveWhitespace(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

// CompactWhitespace replaces multiple consecutive whitespace with single space
func CompactWhitespace(s string) string {
	re := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(re.ReplaceAllString(s, " "))
}

// ============================================================================
// Sensitive Data Masking Functions
// ============================================================================

// MaskSensitive masks sensitive data, showing only first and last few characters
// Example: MaskSensitive("1234567890", 2, 2) -> "12****90"
func MaskSensitive(s string, showFirst, showLast int) string {
	runes := []rune(s)
	length := len(runes)

	// If string is too short, mask everything
	if length <= showFirst+showLast {
		return strings.Repeat("*", length)
	}

	masked := make([]rune, length)

	// Show first characters
	for i := 0; i < showFirst; i++ {
		masked[i] = runes[i]
	}

	// Mask middle characters
	for i := showFirst; i < length-showLast; i++ {
		masked[i] = '*'
	}

	// Show last characters
	for i := length - showLast; i < length; i++ {
		masked[i] = runes[i]
	}

	return string(masked)
}

// MaskEmail masks email address
// Example: "user@example.com" -> "u***@example.com"
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return MaskSensitive(email, 1, 0)
	}

	localPart := parts[0]
	domain := parts[1]

	if len(localPart) <= 1 {
		return localPart + "@" + domain
	}

	maskedLocal := string(localPart[0]) + "***"
	return maskedLocal + "@" + domain
}

// MaskPhone masks phone number
// Example: "1234567890" -> "123***7890"
func MaskPhone(phone string) string {
	// Remove all non-digit characters
	digits := regexp.MustCompile(`\D`).ReplaceAllString(phone, "")

	if len(digits) <= 6 {
		return MaskSensitive(digits, 1, 1)
	}

	return MaskSensitive(digits, 3, 4)
}

// MaskCreditCard masks credit card number
// Example: "1234567890123456" -> "1234********3456"
func MaskCreditCard(cardNumber string) string {
	digits := regexp.MustCompile(`\D`).ReplaceAllString(cardNumber, "")
	return MaskSensitive(digits, 4, 4)
}

// ============================================================================
// Random String Generation Functions
// ============================================================================

// RandomString generates a random alphanumeric string of specified length
func RandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	return randomStringFromCharset(length, charset)
}

// RandomAlphabetic generates a random alphabetic string
func RandomAlphabetic(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	return randomStringFromCharset(length, charset)
}

// RandomNumeric generates a random numeric string
func RandomNumeric(length int) (string, error) {
	const charset = "0123456789"
	return randomStringFromCharset(length, charset)
}

// RandomHex generates a random hexadecimal string
func RandomHex(length int) (string, error) {
	const charset = "0123456789abcdef"
	return randomStringFromCharset(length, charset)
}

// RandomBase64 generates a random base64-encoded string
func RandomBase64(length int) (string, error) {
	// Calculate byte length needed for desired output length
	byteLength := (length*6 + 7) / 8

	bytes := make([]byte, byteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	encoded := base64.URLEncoding.EncodeToString(bytes)

	// Truncate to desired length
	if len(encoded) > length {
		encoded = encoded[:length]
	}

	return encoded, nil
}

// randomStringFromCharset generates random string from custom charset
func randomStringFromCharset(length int, charset string) (string, error) {
	if length <= 0 {
		return "", nil
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	charsetLen := len(charset)
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		result[i] = charset[int(bytes[i])%charsetLen]
	}

	return string(result), nil
}

// ============================================================================
// String Validation Functions
// ============================================================================

// IsEmpty checks if string is empty or contains only whitespace
func IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

// IsNotEmpty checks if string is not empty
func IsNotEmpty(s string) bool {
	return !IsEmpty(s)
}

// Contains checks if string contains substring (case-insensitive)
func ContainsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// HasPrefix checks if string has prefix (case-insensitive)
func HasPrefixIgnoreCase(s, prefix string) bool {
	return strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix))
}

// HasSuffix checks if string has suffix (case-insensitive)
func HasSuffixIgnoreCase(s, suffix string) bool {
	return strings.HasSuffix(strings.ToLower(s), strings.ToLower(suffix))
}

// IsAlphanumeric checks if string contains only alphanumeric characters
func IsAlphanumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// IsAlphabetic checks if string contains only alphabetic characters
func IsAlphabetic(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// IsNumeric checks if string contains only numeric characters
func IsNumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// ============================================================================
// String Formatting Functions
// ============================================================================

// PadLeft pads string on the left to reach specified length
func PadLeft(s string, length int, padChar rune) string {
	runes := []rune(s)
	if len(runes) >= length {
		return s
	}

	padding := strings.Repeat(string(padChar), length-len(runes))
	return padding + s
}

// PadRight pads string on the right to reach specified length
func PadRight(s string, length int, padChar rune) string {
	runes := []rune(s)
	if len(runes) >= length {
		return s
	}

	padding := strings.Repeat(string(padChar), length-len(runes))
	return s + padding
}

// Center centers string within specified length
func Center(s string, length int, padChar rune) string {
	runes := []rune(s)
	currentLen := len(runes)

	if currentLen >= length {
		return s
	}

	totalPad := length - currentLen
	leftPad := totalPad / 2
	rightPad := totalPad - leftPad

	return strings.Repeat(string(padChar), leftPad) + s + strings.Repeat(string(padChar), rightPad)
}

// Indent indents each line of text with specified prefix
func Indent(s string, indent string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}

// WrapText wraps text to specified line length
func WrapText(s string, width int) string {
	if width <= 0 {
		return s
	}

	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	var currentLine strings.Builder

	for _, word := range words {
		if currentLine.Len() == 0 {
			currentLine.WriteString(word)
		} else if currentLine.Len()+1+len(word) <= width {
			currentLine.WriteString(" ")
			currentLine.WriteString(word)
		} else {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
		}
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return strings.Join(lines, "\n")
}

//Personal.AI order the ending
