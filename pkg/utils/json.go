// Package utils provides JSON utility functions for OpenEAAP.
// It includes serialization/deserialization, pretty printing, field filtering,
// and safe JSON operations for API responses and logging.
package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
)

// ============================================================================
// JSON Serialization/Deserialization Functions
// ============================================================================

// ToJSON converts any value to JSON string
func ToJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// ToJSONBytes converts any value to JSON bytes
func ToJSONBytes(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return data, nil
}

// FromJSON parses JSON string into target value
func FromJSON(jsonStr string, target interface{}) error {
	if err := json.Unmarshal([]byte(jsonStr), target); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return nil
}

// FromJSONBytes parses JSON bytes into target value
func FromJSONBytes(data []byte, target interface{}) error {
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return nil
}

// FromJSONReader parses JSON from reader into target value
func FromJSONReader(reader io.Reader, target interface{}) error {
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}
	return nil
}

// ============================================================================
// Pretty Print Functions
// ============================================================================

// PrettyJSON formats JSON string with indentation
func PrettyJSON(jsonStr string) (string, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(jsonStr), "", "  "); err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}
	return buf.String(), nil
}

// PrettyPrint converts value to pretty-printed JSON string
func PrettyPrint(v interface{}) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to pretty print: %w", err)
	}
	return string(data), nil
}

// PrettyPrintBytes converts value to pretty-printed JSON bytes
func PrettyPrintBytes(v interface{}) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to pretty print: %w", err)
	}
	return data, nil
}

// CompactJSON removes whitespace from JSON string
func CompactJSON(jsonStr string) (string, error) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(jsonStr)); err != nil {
		return "", fmt.Errorf("failed to compact JSON: %w", err)
	}
	return buf.String(), nil
}

// ============================================================================
// Field Filtering Functions
// ============================================================================

// FilterFields filters JSON object to include only specified fields
func FilterFields(jsonStr string, fields []string) (string, error) {
	var data map[string]interface{}
	if err := FromJSON(jsonStr, &data); err != nil {
		return "", err
	}

	filtered := make(map[string]interface{})
	for _, field := range fields {
		if val, ok := data[field]; ok {
			filtered[field] = val
		}
	}

	return ToJSON(filtered)
}

// ExcludeFields filters JSON object to exclude specified fields
func ExcludeFields(jsonStr string, fields []string) (string, error) {
	var data map[string]interface{}
	if err := FromJSON(jsonStr, &data); err != nil {
		return "", err
	}

	excludeMap := make(map[string]bool)
	for _, field := range fields {
		excludeMap[field] = true
	}

	filtered := make(map[string]interface{})
	for key, val := range data {
		if !excludeMap[key] {
			filtered[key] = val
		}
	}

	return ToJSON(filtered)
}

// FilterFieldsNested filters nested JSON fields using dot notation
// Example: FilterFieldsNested(json, []string{"user.name", "user.email"})
func FilterFieldsNested(jsonStr string, fields []string) (string, error) {
	var data map[string]interface{}
	if err := FromJSON(jsonStr, &data); err != nil {
		return "", err
	}

	filtered := make(map[string]interface{})

	for _, field := range fields {
		parts := strings.Split(field, ".")
		setNestedField(filtered, parts, getNestedField(data, parts))
	}

	return ToJSON(filtered)
}

// getNestedField retrieves nested field value using path parts
func getNestedField(data map[string]interface{}, parts []string) interface{} {
	if len(parts) == 0 {
		return nil
	}

	val, ok := data[parts[0]]
	if !ok {
		return nil
	}

	if len(parts) == 1 {
		return val
	}

	if nested, ok := val.(map[string]interface{}); ok {
		return getNestedField(nested, parts[1:])
	}

	return nil
}

// setNestedField sets nested field value using path parts
func setNestedField(data map[string]interface{}, parts []string, value interface{}) {
	if len(parts) == 0 || value == nil {
		return
	}

	if len(parts) == 1 {
		data[parts[0]] = value
		return
	}

	if _, ok := data[parts[0]]; !ok {
		data[parts[0]] = make(map[string]interface{})
	}

	if nested, ok := data[parts[0]].(map[string]interface{}); ok {
		setNestedField(nested, parts[1:], value)
	}
}

// ============================================================================
// Safe JSON Operations (Handle Sensitive Data)
// ============================================================================

// MaskSensitiveFields masks sensitive fields in JSON
func MaskSensitiveFields(jsonStr string, sensitiveFields []string) (string, error) {
	var data map[string]interface{}
	if err := FromJSON(jsonStr, &data); err != nil {
		return "", err
	}

	for _, field := range sensitiveFields {
		if _, ok := data[field]; ok {
			data[field] = "***MASKED***"
		}
	}

	return ToJSON(data)
}

// RedactSensitiveFields removes sensitive fields from JSON
func RedactSensitiveFields(jsonStr string, sensitiveFields []string) (string, error) {
	var data map[string]interface{}
	if err := FromJSON(jsonStr, &data); err != nil {
		return "", err
	}

	for _, field := range sensitiveFields {
		delete(data, field)
	}

	return ToJSON(data)
}

// SafeToJSON converts value to JSON, masking sensitive fields
func SafeToJSON(v interface{}, sensitiveFields []string) (string, error) {
	jsonStr, err := ToJSON(v)
	if err != nil {
		return "", err
	}

	return MaskSensitiveFields(jsonStr, sensitiveFields)
}

// ============================================================================
// JSON Validation Functions
// ============================================================================

// IsValidJSON checks if string is valid JSON
func IsValidJSON(jsonStr string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(jsonStr), &js) == nil
}

// IsJSONObject checks if string is a valid JSON object
func IsJSONObject(jsonStr string) bool {
	var obj map[string]interface{}
	return json.Unmarshal([]byte(jsonStr), &obj) == nil
}

// IsJSONArray checks if string is a valid JSON array
func IsJSONArray(jsonStr string) bool {
	var arr []interface{}
	return json.Unmarshal([]byte(jsonStr), &arr) == nil
}

// ValidateJSONSchema validates JSON against expected structure
// This is a simple validation - for complex schemas, use a dedicated library
func ValidateJSONSchema(jsonStr string, requiredFields []string) error {
	var data map[string]interface{}
	if err := FromJSON(jsonStr, &data); err != nil {
		return err
	}

	for _, field := range requiredFields {
		if _, ok := data[field]; !ok {
			return fmt.Errorf("required field missing: %s", field)
		}
	}

	return nil
}

// ============================================================================
// JSON Transformation Functions
// ============================================================================

// MergeJSON merges multiple JSON objects into one
// Later values override earlier ones for duplicate keys
func MergeJSON(jsonStrs ...string) (string, error) {
	merged := make(map[string]interface{})

	for _, jsonStr := range jsonStrs {
		var data map[string]interface{}
		if err := FromJSON(jsonStr, &data); err != nil {
			return "", err
		}

		for key, val := range data {
			merged[key] = val
		}
	}

	return ToJSON(merged)
}

// DeepMergeJSON recursively merges JSON objects
func DeepMergeJSON(jsonStrs ...string) (string, error) {
	merged := make(map[string]interface{})

	for _, jsonStr := range jsonStrs {
		var data map[string]interface{}
		if err := FromJSON(jsonStr, &data); err != nil {
			return "", err
		}

		deepMerge(merged, data)
	}

	return ToJSON(merged)
}

// deepMerge recursively merges source into destination
func deepMerge(dst, src map[string]interface{}) {
	for key, srcVal := range src {
		if dstVal, ok := dst[key]; ok {
			// Both are maps - merge recursively
			if dstMap, dstOk := dstVal.(map[string]interface{}); dstOk {
				if srcMap, srcOk := srcVal.(map[string]interface{}); srcOk {
					deepMerge(dstMap, srcMap)
					continue
				}
			}
		}
		// Override with source value
		dst[key] = srcVal
	}
}

// FlattenJSON flattens nested JSON to single level with dot notation
func FlattenJSON(jsonStr string) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := FromJSON(jsonStr, &data); err != nil {
		return nil, err
	}

	flattened := make(map[string]interface{})
	flattenRecursive(data, "", flattened)
	return flattened, nil
}

// flattenRecursive recursively flattens nested maps
func flattenRecursive(data map[string]interface{}, prefix string, result map[string]interface{}) {
	for key, val := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		if nested, ok := val.(map[string]interface{}); ok {
			flattenRecursive(nested, fullKey, result)
		} else {
			result[fullKey] = val
		}
	}
}

// ============================================================================
// JSON Conversion Utilities
// ============================================================================

// StructToMap converts struct to map[string]interface{} using JSON tags
func StructToMap(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal struct: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to map: %w", err)
	}

	return result, nil
}

// MapToStruct converts map to struct using JSON tags
func MapToStruct(m map[string]interface{}, target interface{}) error {
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal map: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal to struct: %w", err)
	}

	return nil
}

// CloneJSON deep clones a JSON-serializable value
func CloneJSON(src interface{}) (interface{}, error) {
	data, err := json.Marshal(src)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal for cloning: %w", err)
	}

	// Determine target type
	var clone interface{}
	srcType := reflect.TypeOf(src)

	if srcType.Kind() == reflect.Ptr {
		clone = reflect.New(srcType.Elem()).Interface()
	} else {
		clone = reflect.New(srcType).Interface()
	}

	if err := json.Unmarshal(data, clone); err != nil {
		return nil, fmt.Errorf("failed to unmarshal clone: %w", err)
	}

	return clone, nil
}

// ============================================================================
// Streaming JSON Functions
// ============================================================================

// StreamJSONArray processes large JSON arrays in streaming fashion
func StreamJSONArray(reader io.Reader, handler func(interface{}) error) error {
	decoder := json.NewDecoder(reader)

	// Read opening bracket
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to read array start: %w", err)
	}

	if delim, ok := token.(json.Delim); !ok || delim != '[' {
		return fmt.Errorf("expected array start, got %v", token)
	}

	// Process array elements
	for decoder.More() {
		var element interface{}
		if err := decoder.Decode(&element); err != nil {
			return fmt.Errorf("failed to decode element: %w", err)
		}

		if err := handler(element); err != nil {
			return fmt.Errorf("handler error: %w", err)
		}
	}

	// Read closing bracket
	token, err = decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to read array end: %w", err)
	}

	if delim, ok := token.(json.Delim); !ok || delim != ']' {
		return fmt.Errorf("expected array end, got %v", token)
	}

	return nil
}

// EncodeJSONStream encodes values to JSON stream
func EncodeJSONStream(writer io.Writer, values []interface{}) error {
	encoder := json.NewEncoder(writer)

	for _, val := range values {
		if err := encoder.Encode(val); err != nil {
			return fmt.Errorf("failed to encode value: %w", err)
		}
	}

	return nil
}

//Personal.AI order the ending
