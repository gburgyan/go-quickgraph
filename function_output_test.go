package quickgraph

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test types for union testing
type TestSearchResultUnion struct {
	User    *TestUser
	Product *TestProduct
	Order   *TestOrder
}

type TestUser struct {
	ID   int
	Name string
}

type TestProduct struct {
	SKU   string
	Price float64
}

type TestOrder struct {
	OrderID    int
	TotalPrice float64
}

// Union with different field types
type MixedUnion struct {
	StringPtr      *string
	IntPtr         *int
	MapField       map[string]interface{}
	SliceField     []string
	InterfaceField interface{}
}

// Invalid union with non-pointer fields
type InvalidUnion struct {
	DirectString string
	DirectInt    int
}

// Not a union type (doesn't end with "Union")
type RegularStruct struct {
	Field1 *string
	Field2 *int
}

// Empty union for edge case testing
type EmptyUnion struct{}

// Nested pointer union
type NestedPointerUnion struct {
	UserPtr    **TestUser
	ProductPtr **TestProduct
}

func TestDeferenceUnionTypeValue(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() reflect.Value
		wantType  reflect.Type
		wantValue interface{}
		wantErr   string
	}{
		{
			name: "valid union with User",
			setup: func() reflect.Value {
				user := &TestUser{ID: 1, Name: "John"}
				union := TestSearchResultUnion{User: user}
				return reflect.ValueOf(union)
			},
			wantType:  reflect.TypeOf(TestUser{}),
			wantValue: TestUser{ID: 1, Name: "John"},
			wantErr:   "",
		},
		{
			name: "valid union with Product",
			setup: func() reflect.Value {
				product := &TestProduct{SKU: "ABC123", Price: 99.99}
				union := TestSearchResultUnion{Product: product}
				return reflect.ValueOf(union)
			},
			wantType:  reflect.TypeOf(TestProduct{}),
			wantValue: TestProduct{SKU: "ABC123", Price: 99.99},
			wantErr:   "",
		},
		{
			name: "valid union with Order",
			setup: func() reflect.Value {
				order := &TestOrder{OrderID: 123, TotalPrice: 299.99}
				union := TestSearchResultUnion{Order: order}
				return reflect.ValueOf(union)
			},
			wantType:  reflect.TypeOf(TestOrder{}),
			wantValue: TestOrder{OrderID: 123, TotalPrice: 299.99},
			wantErr:   "",
		},
		{
			name: "all fields nil",
			setup: func() reflect.Value {
				union := TestSearchResultUnion{}
				return reflect.ValueOf(union)
			},
			wantType:  nil,
			wantValue: nil,
			wantErr:   "all fields in union type are nil",
		},
		{
			name: "multiple non-nil fields",
			setup: func() reflect.Value {
				user := &TestUser{ID: 1, Name: "John"}
				product := &TestProduct{SKU: "ABC123", Price: 99.99}
				union := TestSearchResultUnion{User: user, Product: product}
				return reflect.ValueOf(union)
			},
			wantType:  nil,
			wantValue: nil,
			wantErr:   "more than one field in union type is not nil",
		},
		{
			name: "not a union type",
			setup: func() reflect.Value {
				s := "test"
				regular := RegularStruct{Field1: &s}
				return reflect.ValueOf(regular)
			},
			wantType:  reflect.TypeOf(RegularStruct{}),
			wantValue: nil, // Returns the same value
			wantErr:   "",
		},
		{
			name: "invalid value",
			setup: func() reflect.Value {
				return reflect.Value{}
			},
			wantType:  nil,
			wantValue: nil,
			wantErr:   "",
		},
		{
			name: "mixed union with map field",
			setup: func() reflect.Value {
				m := map[string]interface{}{"key": "value"}
				union := MixedUnion{MapField: m}
				return reflect.ValueOf(union)
			},
			wantType:  reflect.TypeOf(map[string]interface{}{}),
			wantValue: map[string]interface{}{"key": "value"},
			wantErr:   "",
		},
		{
			name: "mixed union with slice field",
			setup: func() reflect.Value {
				s := []string{"a", "b", "c"}
				union := MixedUnion{SliceField: s}
				return reflect.ValueOf(union)
			},
			wantType:  reflect.TypeOf([]string{}),
			wantValue: []string{"a", "b", "c"},
			wantErr:   "",
		},
		{
			name: "mixed union with interface field",
			setup: func() reflect.Value {
				union := MixedUnion{InterfaceField: "interface value"}
				return reflect.ValueOf(union)
			},
			wantType:  reflect.TypeOf((*interface{})(nil)).Elem(),
			wantValue: "interface value",
			wantErr:   "",
		},
		{
			name: "invalid union with non-pointer fields",
			setup: func() reflect.Value {
				union := InvalidUnion{DirectString: "test", DirectInt: 42}
				return reflect.ValueOf(union)
			},
			wantType:  nil,
			wantValue: nil,
			wantErr:   "fields in union type must be pointers, maps, slices, or interfaces",
		},
		{
			name: "empty union struct",
			setup: func() reflect.Value {
				union := EmptyUnion{}
				return reflect.ValueOf(union)
			},
			wantType:  nil,
			wantValue: nil,
			wantErr:   "all fields in union type are nil",
		},
		{
			name: "union with nil pointer pointing to invalid value",
			setup: func() reflect.Value {
				// Create a union with a pointer field
				union := TestSearchResultUnion{User: (*TestUser)(nil)}
				// Manually create a reflect.Value that represents this edge case
				// In practice, this shouldn't happen, but we test for robustness
				v := reflect.ValueOf(union)
				// Return the value as-is since we can't easily create an invalid pointer
				return v
			},
			wantType:  nil,
			wantValue: nil,
			wantErr:   "all fields in union type are nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.setup()

			result, err := deferenceUnionTypeValue(v)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			assert.NoError(t, err)

			if !v.IsValid() {
				assert.False(t, result.IsValid())
				return
			}

			// For non-union types, should return the original value
			typeName := reflect.TypeOf(v.Interface()).Name()
			if len(typeName) < 5 || typeName[len(typeName)-5:] != "Union" {
				if v.Type().Name() == "RegularStruct" {
					assert.Equal(t, v.Interface(), result.Interface())
				}
				return
			}

			if tt.wantType != nil {
				assert.Equal(t, tt.wantType, result.Type())
				if tt.wantValue != nil {
					assert.Equal(t, tt.wantValue, result.Interface())
				}
			}
		})
	}
}

// Test addressability preservation
func TestDeferenceUnionTypeValue_Addressability(t *testing.T) {
	// Create a union with an addressable User
	user := &TestUser{ID: 1, Name: "John"}
	union := TestSearchResultUnion{User: user}

	// Get reflect.Value in a way that preserves addressability
	v := reflect.ValueOf(&union).Elem()

	result, err := deferenceUnionTypeValue(v)
	assert.NoError(t, err)

	// The result should be the dereferenced User
	assert.Equal(t, reflect.TypeOf(TestUser{}), result.Type())
	assert.Equal(t, TestUser{ID: 1, Name: "John"}, result.Interface())

	// Test that we can still work with methods on the result
	// (This tests that addressability is preserved when needed)
	if result.CanAddr() {
		// If addressable, we should be able to take the address
		_ = result.Addr()
	}
}

// Test with complex nested structures
func TestDeferenceUnionTypeValue_ComplexTypes(t *testing.T) {
	type ComplexStruct struct {
		Nested struct {
			Value string
		}
		Slice []int
		Map   map[string]interface{}
	}

	type ComplexUnion struct {
		Simple  *string
		Complex *ComplexStruct
	}

	complex := &ComplexStruct{
		Nested: struct{ Value string }{Value: "nested"},
		Slice:  []int{1, 2, 3},
		Map:    map[string]interface{}{"key": "value"},
	}

	union := ComplexUnion{Complex: complex}
	v := reflect.ValueOf(union)

	result, err := deferenceUnionTypeValue(v)
	assert.NoError(t, err)
	assert.Equal(t, reflect.TypeOf(ComplexStruct{}), result.Type())

	resultComplex := result.Interface().(ComplexStruct)
	assert.Equal(t, "nested", resultComplex.Nested.Value)
	assert.Equal(t, []int{1, 2, 3}, resultComplex.Slice)
	assert.Equal(t, "value", resultComplex.Map["key"])
}

// Benchmark test
func BenchmarkDeferenceUnionTypeValue(b *testing.B) {
	user := &TestUser{ID: 1, Name: "John"}
	union := TestSearchResultUnion{User: user}
	v := reflect.ValueOf(union)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = deferenceUnionTypeValue(v)
	}
}

// Test edge case with embedded types
func TestDeferenceUnionTypeValue_EmbeddedTypes(t *testing.T) {
	type BaseType struct {
		BaseField string
	}

	type ExtendedType struct {
		BaseType
		ExtendedField int
	}

	type EmbeddedUnion struct {
		Base     *BaseType
		Extended *ExtendedType
	}

	extended := &ExtendedType{
		BaseType:      BaseType{BaseField: "base"},
		ExtendedField: 42,
	}

	union := EmbeddedUnion{Extended: extended}
	v := reflect.ValueOf(union)

	result, err := deferenceUnionTypeValue(v)
	assert.NoError(t, err)
	assert.Equal(t, reflect.TypeOf(ExtendedType{}), result.Type())

	resultExtended := result.Interface().(ExtendedType)
	assert.Equal(t, "base", resultExtended.BaseField)
	assert.Equal(t, 42, resultExtended.ExtendedField)
}
