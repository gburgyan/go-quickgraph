package quickgraph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test types for multi-level embedding navigation
type Level1 struct {
	Value string
}

func (l Level1) GetValue() string {
	return l.Value
}

func (l *Level1) GetValuePtr() string {
	return l.Value
}

type Level2 struct {
	Level1
	Name string
}

func (l Level2) GetName() string {
	return l.Name
}

type Level3 struct {
	Level2
	ID string
}

func (l Level3) GetID() string {
	return l.ID
}

// Test types for pointer embedding
type BasePointer struct {
	BaseValue string
}

func (b BasePointer) GetBaseValue() string {
	return b.BaseValue
}

func (b *BasePointer) GetBaseValuePtr() string {
	return b.BaseValue
}

type MiddlePointer struct {
	*BasePointer
	MiddleValue string
}

func (m MiddlePointer) GetMiddleValue() string {
	return m.MiddleValue
}

type TopPointer struct {
	MiddlePointer
	TopValue string
}

func (t TopPointer) GetTopValue() string {
	return t.TopValue
}

// Test types for mixed pointer/value embedding
type MixedBase struct {
	BaseVal string
}

func (m MixedBase) GetBase() string {
	return m.BaseVal
}

type MixedMiddle struct {
	*MixedBase // pointer embedding
	MiddleVal  string
}

func (m MixedMiddle) GetMiddle() string {
	return m.MiddleVal
}

type MixedTop struct {
	MixedMiddle // value embedding
	TopVal      string
}

func (m MixedTop) GetTop() string {
	return m.TopVal
}

// Test types for error conditions
type NonStruct int

type WithNonStructField struct {
	Field NonStruct
}

type WithNilPointer struct {
	Pointer *Level1
}

type InvalidFieldIndex struct {
	OnlyField string
}

// TestFetchGraphFunction_MultiLevelEmbedding tests navigation through multiple levels of value embedding
func TestFetchGraphFunction_MultiLevelEmbedding(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	g.RegisterQuery(ctx, "GetLevel3", func() Level3 {
		return Level3{
			Level2: Level2{
				Level1: Level1{Value: "nested"},
				Name:   "level2name",
			},
			ID: "level3id",
		}
	})

	// Test accessing method from deeply embedded type
	query := `{
		GetLevel3 {
			GetValue
			GetValuePtr
			GetName
			GetID
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)

	expected := `{"data":{"GetLevel3":{"GetID":"level3id","GetName":"level2name","GetValue":"nested","GetValuePtr":"nested"}}}`
	assert.Equal(t, expected, result)
}

// TestFetchGraphFunction_PointerEmbedding tests navigation through pointer embedding
func TestFetchGraphFunction_PointerEmbedding(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	g.RegisterQuery(ctx, "GetTopPointer", func() TopPointer {
		return TopPointer{
			MiddlePointer: MiddlePointer{
				BasePointer: &BasePointer{BaseValue: "base"},
				MiddleValue: "middle",
			},
			TopValue: "top",
		}
	})

	query := `{
		GetTopPointer {
			GetBaseValue
			GetBaseValuePtr
			GetMiddleValue
			GetTopValue
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)

	expected := `{"data":{"GetTopPointer":{"GetBaseValue":"base","GetBaseValuePtr":"base","GetMiddleValue":"middle","GetTopValue":"top"}}}`
	assert.Equal(t, expected, result)
}

// TestFetchGraphFunction_MixedEmbedding tests navigation through mixed pointer/value embedding
func TestFetchGraphFunction_MixedEmbedding(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	g.RegisterQuery(ctx, "GetMixedTop", func() MixedTop {
		return MixedTop{
			MixedMiddle: MixedMiddle{
				MixedBase: &MixedBase{BaseVal: "base"},
				MiddleVal: "middle",
			},
			TopVal: "top",
		}
	})

	query := `{
		GetMixedTop {
			GetBase
			GetMiddle
			GetTop
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)

	expected := `{"data":{"GetMixedTop":{"GetBase":"base","GetMiddle":"middle","GetTop":"top"}}}`
	assert.Equal(t, expected, result)
}

// TestFetchGraphFunction_NilPointerError tests error handling for nil pointer during navigation
func TestFetchGraphFunction_NilPointerError(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	g.RegisterQuery(ctx, "GetWithNilPointer", func() WithNilPointer {
		return WithNilPointer{Pointer: nil}
	})

	// The library handles nil pointers gracefully by returning null
	query := `{
		GetWithNilPointer {
			Pointer {
				GetValue
			}
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)
	// The result should contain null for the nil pointer
	expected := `{"data":{"GetWithNilPointer":{"Pointer":null}}}`
	assert.Equal(t, expected, result)
}

// Deep embedding test types - defined at package level
type Deep1 struct{ Val string }

func (d Deep1) Get1() string { return d.Val }

type Deep2 struct{ Deep1 }

func (d Deep2) Get2() string { return "level2" }

type Deep3 struct{ Deep2 }

func (d Deep3) Get3() string { return "level3" }

type Deep4 struct{ Deep3 }
func (d Deep4) Get4() string { return "level4" }

type Deep5 struct{ Deep4 }
func (d Deep5) Get5() string { return "level5" }

// TestFetchGraphFunction_DeepEmbeddingPerformance tests performance with deeply nested embedding
func TestFetchGraphFunction_DeepEmbeddingPerformance(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	g.RegisterQuery(ctx, "GetDeep", func() Deep5 {
		return Deep5{
			Deep4: Deep4{
				Deep3: Deep3{
					Deep2: Deep2{
						Deep1: Deep1{Val: "deep"},
					},
				},
			},
		}
	})

	query := `{
		GetDeep {
			Get1
			Get2
			Get3
			Get4
			Get5
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)
	expected := `{"data":{"GetDeep":{"Get1":"deep","Get2":"level2","Get3":"level3","Get4":"level4","Get5":"level5"}}}`
	assert.Equal(t, expected, result)
}

// Field embedding test types - defined at package level  
type BaseWithField struct {
	PublicField string
	Value       string
}

func (b BaseWithField) GetValue() string {
	return b.Value
}

type WrapperWithField struct {
	BaseWithField
	WrapperField string
}

func (w WrapperWithField) GetWrapper() string {
	return w.WrapperField
}

// TestFetchGraphFunction_EmbeddingWithFields tests that both embedded methods and direct fields work
func TestFetchGraphFunction_EmbeddingWithFields(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	g.RegisterQuery(ctx, "GetWrapper", func() WrapperWithField {
		return WrapperWithField{
			BaseWithField: BaseWithField{
				PublicField: "public",
				Value:       "base",
			},
			WrapperField: "wrapper",
		}
	})

	query := `{
		GetWrapper {
			PublicField
			GetValue
			WrapperField
			GetWrapper
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)

	expected := `{"data":{"GetWrapper":{"GetValue":"base","GetWrapper":"wrapper","PublicField":"public","WrapperField":"wrapper"}}}`
	assert.Equal(t, expected, result)
}

// Multiple embedded types test types - defined at package level
type MultiTypeA struct {
	ValueA string
}

func (a MultiTypeA) GetA() string {
	return a.ValueA
}

type MultiTypeB struct {
	ValueB string
}

func (b MultiTypeB) GetB() string {
	return b.ValueB
}

type MultiCombined struct {
	MultiTypeA
	MultiTypeB
	Own string
}

func (c MultiCombined) GetOwn() string {
	return c.Own
}

// TestFetchGraphFunction_MultipleEmbeddedTypes tests a struct with multiple embedded types
func TestFetchGraphFunction_MultipleEmbeddedTypes(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}
	
	g.RegisterQuery(ctx, "GetCombined", func() MultiCombined {
		return MultiCombined{
			MultiTypeA: MultiTypeA{ValueA: "a"},
			MultiTypeB: MultiTypeB{ValueB: "b"},
			Own:        "own",
		}
	})

	query := `{
		GetCombined {
			GetA
			GetB
			GetOwn
			ValueA
			ValueB
			Own
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)

	expected := `{"data":{"GetCombined":{"GetA":"a","GetB":"b","GetOwn":"own","Own":"own","ValueA":"a","ValueB":"b"}}}`
	assert.Equal(t, expected, result)
}

// Interface implementation test types - defined at package level
type NamedInterface interface {
	GetName() string
}

type PersonStruct struct {
	Name string
}

func (p PersonStruct) GetName() string {
	return p.Name
}

type EmployeeStruct struct {
	PersonStruct
	Title string
}

func (e EmployeeStruct) GetTitle() string {
	return e.Title
}

// TestFetchGraphFunction_EmbeddedInterfaceImplementation tests embedded types that implement interfaces
func TestFetchGraphFunction_EmbeddedInterfaceImplementation(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	g.RegisterQuery(ctx, "GetEmployee", func() EmployeeStruct {
		return EmployeeStruct{
			PersonStruct: PersonStruct{Name: "John"},
			Title:        "Developer",
		}
	})

	query := `{
		GetEmployee {
			GetName
			GetTitle
			Name
			Title
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)

	expected := `{"data":{"GetEmployee":{"GetName":"John","GetTitle":"Developer","Name":"John","Title":"Developer"}}}`
	assert.Equal(t, expected, result)
}

// ERROR CONDITION TEST TYPES - defined at package level

// Types for nil pointer navigation test
type ErrorInner struct {
	Value string
}

func (i ErrorInner) GetValue() string {
	return i.Value
}

// Types for embedded pointer to nil test
type ErrorOuter struct {
	Inner *ErrorInner
}

// Types for embedded pointer to nil test  
type ErrorBase struct {
	Value string
}

func (b ErrorBase) GetValue() string {
	return b.Value
}

type ErrorContainer struct {
	*ErrorBase // embedded pointer
}

// ERROR CONDITION TESTS for fieldIndexes loop

// TestFetchGraphFunction_ErrorConditions tests various error scenarios in fieldIndexes navigation
func TestFetchGraphFunction_ErrorConditions(t *testing.T) {
	ctx := context.Background()

	t.Run("NilPointerDuringNavigation", func(t *testing.T) {
		g := Graphy{}

		// This creates a situation where we have a nil pointer in the embedded chain
		g.RegisterQuery(ctx, "GetOuter", func() ErrorOuter {
			return ErrorOuter{Inner: nil}
		})

		query := `{
			GetOuter {
				Inner {
					GetValue
				}
			}
		}`

		result, err := g.ProcessRequest(ctx, query, "")
		// The library handles nil pointers gracefully
		assert.NoError(t, err)
		expected := `{"data":{"GetOuter":{"Inner":null}}}`
		assert.Equal(t, expected, result)
	})

	t.Run("EmbeddedPointerToNil", func(t *testing.T) {
		g := Graphy{}

		// This creates a struct with a nil embedded pointer
		g.RegisterQuery(ctx, "GetContainer", func() ErrorContainer {
			return ErrorContainer{ErrorBase: nil}
		})

		// When we try to call a method on the embedded type, it panics since it's nil
		query := `{
			GetContainer {
				GetValue
			}
		}`

		result, err := g.ProcessRequest(ctx, query, "")
		// Should return an error for calling method on nil pointer
		assert.Error(t, err)
		assert.Contains(t, result, "nil *ErrorBase pointer")
	})
}

// TestFetchGraphFunction_ReflectionErrorPaths tests error paths that are hard to trigger through normal GraphQL usage
func TestFetchGraphFunction_ReflectionErrorPaths(t *testing.T) {
	// Test helper function to create scenarios that would trigger specific error paths
	// These test internal error conditions by manipulating reflection values
	// Most of these are defensive programming checks that are difficult to 
	// trigger through normal GraphQL usage since the type system prevents
	// many of these scenarios.
	
	// Currently this is a placeholder - specific error path tests would need
	// to be implemented here if we want to test internal error conditions
}

// Pointer navigation test types - defined at package level
type PointerInner struct {
	Value string
}

func (i PointerInner) GetValue() string {
	return i.Value
}

func (i *PointerInner) GetValuePtr() string {
	return i.Value
}

type PointerContainer struct {
	InnerPtr *PointerInner
}

// TestFetchGraphFunction_PointerToStructNavigation specifically tests pointer dereferencing
func TestFetchGraphFunction_PointerToStructNavigation(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	g.RegisterQuery(ctx, "GetPointerContainer", func() PointerContainer {
		return PointerContainer{
			InnerPtr: &PointerInner{Value: "pointer-value"},
		}
	})

	query := `{
		GetPointerContainer {
			InnerPtr {
				GetValue
				GetValuePtr
				Value
			}
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)
	expected := `{"data":{"GetPointerContainer":{"InnerPtr":{"GetValue":"pointer-value","GetValuePtr":"pointer-value","Value":"pointer-value"}}}}`
	assert.Equal(t, expected, result)
}


// Complex embedding test types - defined at package level
type ComplexA struct {
	AValue string
	BValue string
}

func (a ComplexA) GetA() string { return a.AValue }

type ComplexB struct {
	*ComplexA // pointer embedding
	BValue    string
	CValue    string
}

func (b ComplexB) GetB() string { return b.BValue }

type ComplexC struct {
	ComplexB // value embedding
	CValue   string
	DValue   string
}

func (c ComplexC) GetC() string { return c.CValue }

type ComplexD struct {
	*ComplexC // pointer embedding again
	DValue    string
}

func (d ComplexD) GetD() string { return d.DValue }

// TestFetchGraphFunction_ComplexEmbeddingChain tests a complex chain of embeddings
func TestFetchGraphFunction_ComplexEmbeddingChain(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	g.RegisterQuery(ctx, "GetComplexD", func() ComplexD {
		return ComplexD{
			ComplexC: &ComplexC{
				ComplexB: ComplexB{
					ComplexA: &ComplexA{AValue: "a"},
					BValue:   "b",
				},
				CValue: "c",
			},
			DValue: "d",
		}
	})

	query := `{
		GetComplexD {
			GetA
			GetB  
			GetC
			GetD
			DValue
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)

	expected := `{"data":{"GetComplexD":{"DValue":"d","GetA":"a","GetB":"b","GetC":"c","GetD":"d"}}}`
	assert.Equal(t, expected, result)
}

// Regression test types - defined at package level
// These types specifically reproduce the original panic condition
type RegressionBase struct {
	Value string
}

func (b RegressionBase) GetValue() string {
	return b.Value
}

type RegressionContainer struct {
	*RegressionBase // This pointer embedding caused the original panic
	ContainerField  string
}

func (c RegressionContainer) GetContainer() string {
	return c.ContainerField
}

// TestRegression_PointerEmbeddingPanic tests the specific regression case
// Bug details:
// - Before fix: panic: reflect: NumField of non-struct type *RegressionBase
// - Root cause: Anonymous pointer fields passed pointer type to populateTypeLookup 
//   instead of dereferencing to struct type first
// - Location: type_lookup.go processBaseTypeFieldLookup function
func TestRegression_PointerEmbeddingPanic(t *testing.T) {
	ctx := context.Background()

	// TEST 1: Ensure type registration doesn't panic (this was the main bug)
	g := Graphy{}
	
	// "panic: reflect: NumField of non-struct type *RegressionBase"
	assert.NotPanics(t, func() {
		g.RegisterQuery(ctx, "GetRegressionContainer", func() RegressionContainer {
			return RegressionContainer{
				RegressionBase:  &RegressionBase{Value: "base"},
				ContainerField: "container",
			}
		})
	}, "Type registration should not panic with pointer embedding")

	// TEST 2: Ensure the registered function actually works
	query := `{
		GetRegressionContainer {
			GetValue
			GetContainer
			ContainerField
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)

	expected := `{"data":{"GetRegressionContainer":{"ContainerField":"container","GetContainer":"container","GetValue":"base"}}}`
	assert.Equal(t, expected, result)

	// TEST 3: Ensure schema generation works (another place that could fail)
	schema := g.SchemaDefinition(ctx)
	assert.Contains(t, schema, "GetRegressionContainer")
	assert.Contains(t, schema, "GetValue")
	assert.Contains(t, schema, "GetContainer")
}