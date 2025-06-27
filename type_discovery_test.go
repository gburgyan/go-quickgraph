package quickgraph

import (
	"testing"
)

// Test types for type discovery

type BaseType struct {
	ID         string
	Name       string
	actualType interface{} `graphy:"-"`
}

func (b *BaseType) ActualType() interface{} {
	if b.actualType != nil {
		return b.actualType
	}
	return b
}

type DerivedTypeA struct {
	BaseType
	FieldA string
}

func NewDerivedTypeA(id, name, fieldA string) *DerivedTypeA {
	d := &DerivedTypeA{
		BaseType: BaseType{ID: id, Name: name},
		FieldA:   fieldA,
	}
	d.BaseType.actualType = d
	return d
}

type DerivedTypeB struct {
	BaseType
	FieldB int
}

func NewDerivedTypeB(id, name string, fieldB int) *DerivedTypeB {
	d := &DerivedTypeB{
		BaseType: BaseType{ID: id, Name: name},
		FieldB:   fieldB,
	}
	d.BaseType.actualType = d
	return d
}

// Type without discovery support
type SimpleType struct {
	Value string
}

func TestDiscoverGeneric(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		wantType interface{}
		wantOk   bool
	}{
		{
			name:     "Discover DerivedTypeA from BaseType pointer",
			input:    &NewDerivedTypeA("1", "Test", "ValueA").BaseType,
			wantType: &DerivedTypeA{},
			wantOk:   true,
		},
		{
			name:     "Discover DerivedTypeB from BaseType pointer",
			input:    &NewDerivedTypeB("2", "Test", 42).BaseType,
			wantType: &DerivedTypeB{},
			wantOk:   true,
		},
		{
			name:     "Direct type assertion for DerivedTypeA",
			input:    NewDerivedTypeA("3", "Test", "ValueA"),
			wantType: &DerivedTypeA{},
			wantOk:   true,
		},
		{
			name:     "Type without discovery support",
			input:    &SimpleType{Value: "test"},
			wantType: &SimpleType{},
			wantOk:   true,
		},
		{
			name:     "Nil input",
			input:    nil,
			wantType: (*DerivedTypeA)(nil),
			wantOk:   false,
		},
		{
			name:     "Wrong type discovery",
			input:    &NewDerivedTypeA("4", "Test", "ValueA").BaseType,
			wantType: &DerivedTypeB{},
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test based on the expected type
			switch tt.wantType.(type) {
			case *DerivedTypeA:
				got, ok := Discover[*DerivedTypeA](tt.input)
				if ok != tt.wantOk {
					t.Errorf("Discover() ok = %v, want %v", ok, tt.wantOk)
				}
				if tt.wantOk && got == nil {
					t.Errorf("Discover() returned nil when expecting a value")
				}
			case *DerivedTypeB:
				got, ok := Discover[*DerivedTypeB](tt.input)
				if ok != tt.wantOk {
					t.Errorf("Discover() ok = %v, want %v", ok, tt.wantOk)
				}
				if tt.wantOk && got == nil {
					t.Errorf("Discover() returned nil when expecting a value")
				}
			case *SimpleType:
				got, ok := Discover[*SimpleType](tt.input)
				if ok != tt.wantOk {
					t.Errorf("Discover() ok = %v, want %v", ok, tt.wantOk)
				}
				if tt.wantOk && got == nil {
					t.Errorf("Discover() returned nil when expecting a value")
				}
			}
		})
	}
}

func TestDiscoverType(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		wantValue interface{}
		wantOk    bool
	}{
		{
			name:      "Discoverable type returns actual type",
			input:     &NewDerivedTypeA("1", "Test", "ValueA").BaseType,
			wantValue: NewDerivedTypeA("1", "Test", "ValueA"),
			wantOk:    true,
		},
		{
			name:      "Non-discoverable type returns self",
			input:     &SimpleType{Value: "test"},
			wantValue: &SimpleType{Value: "test"},
			wantOk:    true,
		},
		{
			name:      "Nil actualType returns self",
			input:     &BaseType{ID: "1", Name: "Test"},
			wantValue: &BaseType{ID: "1", Name: "Test"},
			wantOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := DiscoverType(tt.input)
			if ok != tt.wantOk {
				t.Errorf("DiscoverType() ok = %v, want %v", ok, tt.wantOk)
			}

			// For discoverable types, check the type matches
			if tt.wantOk {
				switch tt.wantValue.(type) {
				case *DerivedTypeA:
					if _, ok := got.(*DerivedTypeA); !ok {
						t.Errorf("DiscoverType() returned wrong type, got %T, want *DerivedTypeA", got)
					}
				case *BaseType:
					if _, ok := got.(*BaseType); !ok {
						t.Errorf("DiscoverType() returned wrong type, got %T, want *BaseType", got)
					}
				case *SimpleType:
					// For non-discoverable types, we just check it returns the same value
					if got != tt.input {
						t.Errorf("DiscoverType() = %v, want %v", got, tt.input)
					}
				}
			}
		})
	}
}

func TestTypeDiscoveryInGraphQLContext(t *testing.T) {
	// Create instances
	devA := NewDerivedTypeA("1", "Alice", "Frontend")
	devB := NewDerivedTypeB("2", "Bob", 5)

	// Simulate returning base type pointers from a function
	employees := []*BaseType{
		&devA.BaseType,
		&devB.BaseType,
	}

	// Test that we can discover the actual types
	for i, emp := range employees {
		if devA, ok := Discover[*DerivedTypeA](emp); ok {
			if i != 0 {
				t.Errorf("Expected DerivedTypeA at index 0, got it at index %d", i)
			}
			if devA.FieldA != "Frontend" {
				t.Errorf("Expected FieldA = Frontend, got %s", devA.FieldA)
			}
		}

		if devB, ok := Discover[*DerivedTypeB](emp); ok {
			if i != 1 {
				t.Errorf("Expected DerivedTypeB at index 1, got it at index %d", i)
			}
			if devB.FieldB != 5 {
				t.Errorf("Expected FieldB = 5, got %d", devB.FieldB)
			}
		}
	}
}

// Benchmark tests
func BenchmarkDiscover(b *testing.B) {
	dev := NewDerivedTypeA("1", "Test", "Value")
	base := &dev.BaseType

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Discover[*DerivedTypeA](base)
	}
}

func BenchmarkDiscoverType(b *testing.B) {
	dev := NewDerivedTypeA("1", "Test", "Value")
	base := &dev.BaseType

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DiscoverType(base)
	}
}

func BenchmarkDirectTypeAssertion(b *testing.B) {
	dev := NewDerivedTypeA("1", "Test", "Value")
	var iface interface{} = dev

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = iface.(*DerivedTypeA)
	}
}
