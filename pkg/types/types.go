// Package types provides common type definitions used across OpenEAAP.
// It defines foundational types like ID, timestamp, pagination, and
// base structures to ensure type consistency throughout the platform.
package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// ID Types
// ============================================================================

// ID represents a unique identifier using UUID v4
type ID string

// NewID generates a new unique ID
func NewID() ID {
	return ID(uuid.New().String())
}

// String returns the string representation of ID
func (id ID) String() string {
	return string(id)
}

// IsZero checks if ID is empty
func (id ID) IsZero() bool {
	return id == ""
}

// Valid checks if ID is a valid UUID
func (id ID) Valid() bool {
	_, err := uuid.Parse(string(id))
	return err == nil
}

// MarshalJSON implements json.Marshaler
func (id ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(id))
}

// UnmarshalJSON implements json.Unmarshaler
func (id *ID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*id = ID(s)
	return nil
}

// Value implements driver.Valuer for database storage
func (id ID) Value() (driver.Value, error) {
	if id.IsZero() {
		return nil, nil
	}
	return string(id), nil
}

// Scan implements sql.Scanner for database retrieval
func (id *ID) Scan(value interface{}) error {
	if value == nil {
		*id = ""
		return nil
	}

	switch v := value.(type) {
	case string:
		*id = ID(v)
	case []byte:
		*id = ID(string(v))
	default:
		return fmt.Errorf("cannot scan type %T into ID", value)
	}
	return nil
}

// ============================================================================
// Timestamp Types
// ============================================================================

// Timestamp represents a point in time with millisecond precision
type Timestamp time.Time

// Now returns the current timestamp
func Now() Timestamp {
	return Timestamp(time.Now())
}

// NewTimestamp creates a Timestamp from time.Time
func NewTimestamp(t time.Time) Timestamp {
	return Timestamp(t)
}

// Time converts Timestamp to time.Time
func (ts Timestamp) Time() time.Time {
	return time.Time(ts)
}

// Unix returns the Unix timestamp in seconds
func (ts Timestamp) Unix() int64 {
	return time.Time(ts).Unix()
}

// UnixMilli returns the Unix timestamp in milliseconds
func (ts Timestamp) UnixMilli() int64 {
	return time.Time(ts).UnixMilli()
}

// String returns RFC3339 formatted string
func (ts Timestamp) String() string {
	return time.Time(ts).Format(time.RFC3339)
}

// IsZero checks if timestamp is zero value
func (ts Timestamp) IsZero() bool {
	return time.Time(ts).IsZero()
}

// Before reports whether ts is before other
func (ts Timestamp) Before(other Timestamp) bool {
	return time.Time(ts).Before(time.Time(other))
}

// After reports whether ts is after other
func (ts Timestamp) After(other Timestamp) bool {
	return time.Time(ts).After(time.Time(other))
}

// MarshalJSON implements json.Marshaler
func (ts Timestamp) MarshalJSON() ([]byte, error) {
	if ts.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(time.Time(ts).Format(time.RFC3339))
}

// UnmarshalJSON implements json.Unmarshaler
func (ts *Timestamp) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" || s == "null" {
		*ts = Timestamp{}
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	*ts = Timestamp(t)
	return nil
}

// Value implements driver.Valuer for database storage
func (ts Timestamp) Value() (driver.Value, error) {
	if ts.IsZero() {
		return nil, nil
	}
	return time.Time(ts), nil
}

// Scan implements sql.Scanner for database retrieval
func (ts *Timestamp) Scan(value interface{}) error {
	if value == nil {
		*ts = Timestamp{}
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		*ts = Timestamp(v)
	default:
		return fmt.Errorf("cannot scan type %T into Timestamp", value)
	}
	return nil
}

// ============================================================================
// Pagination Types
// ============================================================================

// PageRequest represents pagination request parameters
type PageRequest struct {
	// Page is the page number (1-indexed)
	Page int `json:"page" form:"page" validate:"min=1"`

	// PageSize is the number of items per page
	PageSize int `json:"page_size" form:"page_size" validate:"min=1,max=100"`

	// SortBy is the field to sort by
	SortBy string `json:"sort_by,omitempty" form:"sort_by"`

	// SortOrder is the sort direction (asc/desc)
	SortOrder SortOrder `json:"sort_order,omitempty" form:"sort_order"`
}

// NewPageRequest creates a default PageRequest
func NewPageRequest() *PageRequest {
	return &PageRequest{
		Page:      1,
		PageSize:  20,
		SortOrder: SortOrderAsc,
	}
}

// Offset calculates the offset for database queries
func (pr *PageRequest) Offset() int {
	return (pr.Page - 1) * pr.PageSize
}

// Limit returns the page size for database queries
func (pr *PageRequest) Limit() int {
	return pr.PageSize
}

// PageResponse represents paginated response metadata
type PageResponse struct {
	// Page is the current page number
	Page int `json:"page"`

	// PageSize is the number of items per page
	PageSize int `json:"page_size"`

	// TotalItems is the total number of items
	TotalItems int64 `json:"total_items"`

	// TotalPages is the total number of pages
	TotalPages int `json:"total_pages"`

	// HasNext indicates if there is a next page
	HasNext bool `json:"has_next"`

	// HasPrev indicates if there is a previous page
	HasPrev bool `json:"has_prev"`
}

// NewPageResponse creates a PageResponse from request and total count
func NewPageResponse(req *PageRequest, totalItems int64) *PageResponse {
	totalPages := int((totalItems + int64(req.PageSize) - 1) / int64(req.PageSize))

	return &PageResponse{
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasNext:    req.Page < totalPages,
		HasPrev:    req.Page > 1,
	}
}

// SortOrder represents sort direction
type SortOrder string

const (
	// SortOrderAsc represents ascending sort order
	SortOrderAsc SortOrder = "asc"

	// SortOrderDesc represents descending sort order
	SortOrderDesc SortOrder = "desc"
)

// Valid checks if sort order is valid
func (so SortOrder) Valid() bool {
	return so == SortOrderAsc || so == SortOrderDesc
}

// ============================================================================
// Filter Types
// ============================================================================

// FilterOperator represents filter comparison operators
type FilterOperator string

const (
	FilterOpEqual        FilterOperator = "eq"
	FilterOpNotEqual     FilterOperator = "ne"
	FilterOpGreaterThan  FilterOperator = "gt"
	FilterOpGreaterEqual FilterOperator = "gte"
	FilterOpLessThan     FilterOperator = "lt"
	FilterOpLessEqual    FilterOperator = "lte"
	FilterOpIn           FilterOperator = "in"
	FilterOpNotIn        FilterOperator = "nin"
	FilterOpContains     FilterOperator = "contains"
	FilterOpStartsWith   FilterOperator = "starts_with"
	FilterOpEndsWith     FilterOperator = "ends_with"
)

// Filter represents a single filter condition
type Filter struct {
	Field    string         `json:"field"`
	Operator FilterOperator `json:"operator"`
	Value    interface{}    `json:"value"`
}

// FilterRequest represents a collection of filters
type FilterRequest struct {
	Filters []Filter `json:"filters,omitempty"`
}

// ============================================================================
// Metadata Types
// ============================================================================

// Metadata represents flexible key-value metadata
type Metadata map[string]interface{}

// Get retrieves a value from metadata
func (m Metadata) Get(key string) (interface{}, bool) {
	val, ok := m[key]
	return val, ok
}

// GetString retrieves a string value from metadata
func (m Metadata) GetString(key string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

// GetInt retrieves an int value from metadata
func (m Metadata) GetInt(key string) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return 0
}

// GetBool retrieves a bool value from metadata
func (m Metadata) GetBool(key string) bool {
	if val, ok := m[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// Set sets a value in metadata
func (m Metadata) Set(key string, value interface{}) {
	m[key] = value
}

// Delete removes a key from metadata
func (m Metadata) Delete(key string) {
	delete(m, key)
}

// ============================================================================
// Status Types
// ============================================================================

// Status represents a generic status
type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusPending  Status = "pending"
	StatusDeleted  Status = "deleted"
)

// Valid checks if status is valid
func (s Status) Valid() bool {
	switch s {
	case StatusActive, StatusInactive, StatusPending, StatusDeleted:
		return true
	default:
		return false
	}
}

// ============================================================================
// Base Entity
// ============================================================================

// BaseEntity provides common fields for all entities
type BaseEntity struct {
	ID        ID        `json:"id" db:"id"`
	CreatedAt Timestamp `json:"created_at" db:"created_at"`
	UpdatedAt Timestamp `json:"updated_at" db:"updated_at"`
	DeletedAt *Timestamp `json:"deleted_at,omitempty" db:"deleted_at"`
}

// NewBaseEntity creates a new BaseEntity with generated ID and timestamps
func NewBaseEntity() BaseEntity {
	now := Now()
	return BaseEntity{
		ID:        NewID(),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsDeleted checks if entity is soft deleted
func (be *BaseEntity) IsDeleted() bool {
	return be.DeletedAt != nil && !be.DeletedAt.IsZero()
}

// MarkDeleted marks entity as soft deleted
func (be *BaseEntity) MarkDeleted() {
	now := Now()
	be.DeletedAt = &now
}

// Touch updates the UpdatedAt timestamp
func (be *BaseEntity) Touch() {
	be.UpdatedAt = Now()
}

// ============================================================================
// Version Type
// ============================================================================

// Version represents a semantic version
type Version struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch"`
}

// String returns version in semantic format (e.g., "1.2.3")
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare compares two versions (-1: less, 0: equal, 1: greater)
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}
	return 0
}

//Personal.AI order the ending
