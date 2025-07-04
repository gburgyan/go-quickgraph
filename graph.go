package quickgraph

import (
	"context"
	"github.com/gburgyan/go-timing"
	"log"
	"reflect"
	"strings"
	"sync"
)

// Graphy is the main entry point for the go-quickgraph library. This holds all the
// registered functions and types and provides methods for executing requests.
// It is safe to use concurrently once it has been initialized -- there is no guarantee
// that the initialization is thread-safe.
//
// The zero value for Graphy is safe to use.
//
// The RequestCache is optional, but it can be used to cache the results of parsing
// requests. This can be useful if you are using the library in a server environment
// and want to cache the results of parsing requests to speed things up. Refer to
// GraphRequestCache for more information.
type Graphy struct {
	RequestCache GraphRequestCache

	EnableTiming bool

	// ProductionMode controls error reporting - when true, sensitive information
	// like stack traces, function names, and inner error details are sanitized
	// to prevent information disclosure. Defaults to false (development mode).
	ProductionMode bool

	// QueryLimits defines optional limits to prevent DoS attacks
	QueryLimits *QueryLimits

	// MemoryLimits defines optional limits to prevent memory exhaustion attacks
	MemoryLimits *MemoryLimits

	// CORSSettings defines CORS configuration for HTTP responses
	// Nil means no CORS headers will be added
	CORSSettings *CORSSettings

	processors  map[string]graphFunction
	typeLookups map[reflect.Type]*typeLookup
	anyTypes    []*typeLookup

	// explicitTypes holds types that were explicitly registered via RegisterType
	explicitTypes []*typeLookup

	// scalars holds registered custom scalar types
	scalars *scalarRegistry

	schemaEnabled bool
	schemaBuffer  *schemaTypes

	// typeMutex is used to ensure that nothing strange happens when multiple threads
	// are trying to add to the typeLookups map at the same time.
	typeMutex sync.Mutex

	// structureLock ensures that there cannot be concurrent modifications to the
	// processors while there are schema-related requests in progress.
	structureLock sync.RWMutex

	// errorHandler for handling errors in a customizable way
	errorHandler ErrorHandler

	// schemaLock ensures that there is only a single schema-generation request in
	// progress at a time.
	schemaLock sync.Mutex
}

type GraphTypeExtension interface {
	GraphTypeExtension() GraphTypeInfo
}

type GraphTypeInfo struct {
	// Name is the name of the type.
	Name string

	// Description is the description of the type.
	Description string

	// Deprecated is the deprecation status of the type.
	Deprecated string

	// Function overrides for the type.
	FunctionDefinitions []FunctionDefinition

	// InterfaceOnly indicates that this type should only generate an interface,
	// not both interface and concrete type when it has implementations.
	InterfaceOnly bool
}

var ignoredFunctions = map[string]bool{
	"GraphTypeExtension": true,
	"ActualType":         true,
}

var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
var errorType = reflect.TypeOf((*error)(nil)).Elem()
var stringType = reflect.TypeOf((*string)(nil)).Elem()
var anyType = reflect.TypeOf((*any)(nil)).Elem()
var graphTypeExtensionType = reflect.TypeOf((*GraphTypeExtension)(nil)).Elem()

// RegisterQuery registers a function as a query.
//
// The function must return a valid result value and may return an error. If the function
// returns multiple values, they must be pointers and the result will be an implicit
// union type.
//
// If the names are specified, they must match the non-context parameter count of the function.
// If the names are not specified, then the parameters are dealt with as either anonymous
// parameters or as a single parameter that is a struct. If the function has a single parameter
// that is a struct, then the names of the struct fields are used as the parameter names.
func (g *Graphy) RegisterQuery(ctx context.Context, name string, f any, names ...string) {
	g.ensureInitialized()
	gf := g.newGraphFunction(FunctionDefinition{
		Name:           name,
		Function:       f,
		ParameterNames: names,
		Mode:           ModeQuery,
	}, false)
	g.processors[name] = gf
}

// RegisterMutation registers a function as a mutator.
//
// The function must return a valid result value and may return an error. If the function
// returns multiple values, they must be pointers and the result will be an implicit
// union type.
//
// If the names are specified, they must match the non-context parameter count of the function.
// If the names are not specified, then the parameters are dealt with as either anonymous
// parameters or as a single parameter that is a struct. If the function has a single parameter
// that is a struct, then the names of the struct fields are used as the parameter names.
func (g *Graphy) RegisterMutation(ctx context.Context, name string, f any, names ...string) {
	g.ensureInitialized()
	gf := g.newGraphFunction(FunctionDefinition{
		Name:           name,
		Function:       f,
		ParameterNames: names,
		Mode:           ModeMutation,
	}, false)
	g.processors[name] = gf
}

// RegisterSubscription registers a function as a subscription.
//
// The function must return a channel that emits values of the subscription type.
// The channel should be closed when the subscription ends. The function may optionally
// return an error as a second return value.
//
// Examples:
//
//	// With error return:
//	g.RegisterSubscription(ctx, "messageAdded", func(ctx context.Context, roomID string) (<-chan Message, error) {
//	    ch := make(chan Message)
//	    // Set up subscription logic here
//	    return ch, nil
//	})
//
//	// Without error return:
//	g.RegisterSubscription(ctx, "timeUpdate", func(ctx context.Context) <-chan time.Time {
//	    ch := make(chan time.Time)
//	    // Set up subscription logic here
//	    return ch
//	})
//
// If the names are specified, they must match the non-context parameter count of the function.
// If the names are not specified, then the parameters are dealt with as either anonymous
// parameters or as a single parameter that is a struct. If the function has a single parameter
// that is a struct, then the names of the struct fields are used as the parameter names.
func (g *Graphy) RegisterSubscription(ctx context.Context, name string, f any, names ...string) {
	g.ensureInitialized()
	gf := g.newGraphFunction(FunctionDefinition{
		Name:           name,
		Function:       f,
		ParameterNames: names,
		Mode:           ModeSubscription,
	}, false)
	g.processors[name] = gf
}

// RegisterFunction is similar to both RegisterQuery and RegisterMutation, but it allows
// the caller to specify additional parameters that are less commonly used. See the
// FunctionDefinition documentation for more information.
func (g *Graphy) RegisterFunction(ctx context.Context, def FunctionDefinition) {
	g.structureLock.Lock()
	defer g.structureLock.Unlock()

	g.ensureInitialized()
	gf := g.newGraphFunction(def, false)
	g.processors[def.Name] = gf

	g.schemaBuffer = nil
}

// RegisterAnyType registers a type that is potentially used as a return type for a function
// that returns `any`. This isn't critical to use all cases, but it can be needed for results
// that contain functions that can be called. Without this, those functions would not be
// found -- this it needed to infer the types of parameters in cases those are fulfilled with
// variables.
func (g *Graphy) RegisterAnyType(ctx context.Context, types ...any) {
	g.structureLock.Lock()
	defer g.structureLock.Unlock()

	for _, t := range types {
		tl := g.typeLookup(reflect.TypeOf(t))
		g.anyTypes = append(g.anyTypes, tl)
	}

	g.schemaBuffer = nil
}

// RegisterTypes is a method on the Graphy struct that registers types that implement interfaces.
// This is useful for discovering types that implement certain interfaces.
// The method takes in a context and a variadic parameter of types (of any kind).
// It iterates over the provided types and performs a type lookup for each type.
//
// Parameters:
// - ctx: The context within which the method operates.
// - types: A variadic parameter that represents instances ot types to be registered.
//
// Usage:
// g := &Graphy{}
// g.RegisterTypes(context.Background(), Type1{}, Type2{}, Type3{})
func (g *Graphy) RegisterTypes(ctx context.Context, types ...any) {
	g.structureLock.Lock()
	defer g.structureLock.Unlock()

	for _, t := range types {
		tl := g.typeLookup(reflect.TypeOf(t))
		// Add to explicit types to ensure they're included in schema generation
		g.explicitTypes = append(g.explicitTypes, tl)
	}

	g.schemaBuffer = nil
}

// SetErrorHandler sets a custom error handler for the Graphy instance
// This allows applications to customize how errors are logged and handled
func (g *Graphy) SetErrorHandler(handler ErrorHandler) {
	g.errorHandler = handler
}

// handleError is an internal helper that calls the registered error handler
// or falls back to default logging if no handler is set
func (g *Graphy) handleError(ctx context.Context, category ErrorCategory, err error, details map[string]interface{}) {
	if g.errorHandler != nil {
		g.errorHandler.HandleError(ctx, category, err, details)
	} else {
		// Default fallback behavior - log to standard logger
		log.Printf("[%s] %v", category, err)
	}
}

func (g *Graphy) ensureInitialized() {
	if g.processors == nil {
		g.processors = map[string]graphFunction{}
	}
}

func (g *Graphy) ProcessRequest(ctx context.Context, request string, variableJson string) (string, error) {
	g.structureLock.RLock()
	defer g.structureLock.RUnlock()

	var tCtx context.Context
	var timingContext *timing.Context
	if g.EnableTiming {
		var complete timing.Complete
		timingContext, complete = timing.Start(ctx, "ProcessGraphRequest")
		tCtx = timingContext
		defer complete()
	} else {
		tCtx = ctx
	}

	rs, err := g.getRequestStub(tCtx, request)
	if err != nil {
		return formatErrorWithMode(g.ProductionMode, err), err
	}

	if timingContext != nil {
		timingContext.AddDetails("request", rs.Name())
	}

	newRequest, err := rs.newRequest(tCtx, variableJson)
	if err != nil {
		return formatErrorWithMode(g.ProductionMode, err), err
	}

	return newRequest.execute(tCtx)
}

func (g *Graphy) typeLookup(typ reflect.Type) *typeLookup {
	g.typeMutex.Lock()

	if g.typeLookups == nil {
		g.typeLookups = map[reflect.Type]*typeLookup{}
	}

	if tl, ok := g.typeLookups[typ]; ok {
		g.typeMutex.Unlock()
		return tl
	}

	result := &typeLookup{
		typ:                 typ,
		fields:              make(map[string]fieldLookup),
		fieldsLowercase:     make(map[string]fieldLookup),
		implements:          make(map[string]*typeLookup),
		implementsLowercase: make(map[string]*typeLookup),
		union:               make(map[string]*typeLookup),
		unionLowercase:      make(map[string]*typeLookup),
	}

	rootTyp := typ

	if rootTyp.Kind() == reflect.Ptr {
		rootTyp = rootTyp.Elem()
		result.isPointer = true
	}
	if rootTyp.Kind() == reflect.Slice {
		rootTyp = rootTyp.Elem()
		rootTyp, result.array = g.dereferenceSlice(rootTyp)
	}

	result.rootType = rootTyp

	// Check rootTyp instead of typ since we've already dereferenced pointers
	if rootTyp.Implements(graphTypeExtensionType) || reflect.PtrTo(rootTyp).Implements(graphTypeExtensionType) {
		var gtei GraphTypeExtension
		if rootTyp.Implements(graphTypeExtensionType) {
			// Type implements the interface directly
			gtev := reflect.New(rootTyp).Elem()
			gtei = gtev.Interface().(GraphTypeExtension)
		} else {
			// Pointer to type implements the interface
			gtev := reflect.New(rootTyp)
			gtei = gtev.Interface().(GraphTypeExtension)
		}
		typeExtension := gtei.GraphTypeExtension()

		// Check if this is likely an embedded implementation by comparing the returned name
		// with the actual type name. If they differ and the returned name matches a field name,
		// it's likely inherited from an embedded type.
		if typeExtension.Name != "" && typeExtension.Name != rootTyp.Name() {
			// Check if this type embeds another type with this name
			hasEmbeddedWithName := false
			if rootTyp.Kind() == reflect.Struct {
				for i := 0; i < rootTyp.NumField(); i++ {
					field := rootTyp.Field(i)
					if field.Anonymous && field.Type.Name() == typeExtension.Name {
						hasEmbeddedWithName = true
						break
					}
				}
			}

			if hasEmbeddedWithName {
				// This GraphTypeExtension is inherited from an embedded field
				// Use the actual type name instead
				result.name = rootTyp.Name()
			} else {
				// This is a legitimate rename via GraphTypeExtension
				result.name = typeExtension.Name
				if typeExtension.Deprecated != "" {
					result.isDeprecated = true
					result.deprecatedReason = typeExtension.Deprecated
				}
				if typeExtension.Description != "" {
					result.description = &typeExtension.Description
				}
				result.interfaceOnly = typeExtension.InterfaceOnly
			}
		} else {
			// Names match or Name is empty, use the actual type name
			if typeExtension.Name == "" {
				result.name = rootTyp.Name()
			} else {
				result.name = typeExtension.Name
			}
			if typeExtension.Deprecated != "" {
				result.isDeprecated = true
				result.deprecatedReason = typeExtension.Deprecated
			}
			if typeExtension.Description != "" {
				result.description = &typeExtension.Description
			}
			result.interfaceOnly = typeExtension.InterfaceOnly
		}
	} else {
		result.name = rootTyp.Name()
		// If the name is empty (anonymous types), try to get it from the package path
		if result.name == "" {
			// For types without a direct name, try to extract from the string representation
			typStr := rootTyp.String()
			// Handle pointer types like "*handlers.Employee"
			typStr = strings.TrimPrefix(typStr, "*")
			// Handle slice types like "[]handlers.Employee"
			typStr = strings.TrimPrefix(typStr, "[]")
			// Handle cases like "handlers.Employee"
			if idx := strings.LastIndex(typStr, "."); idx != -1 {
				result.name = typStr[idx+1:]
			} else {
				result.name = typStr
			}
		}
	}

	// Check if this type is registered as a scalar first
	if scalar, isScalar := g.GetScalarByType(rootTyp); isScalar {
		result.fundamental = true
		result.name = scalar.Name // Use the scalar name instead of the Go type name
		g.typeLookups[typ] = result
		g.typeMutex.Unlock()
		return result
	}

	if rootTyp.Kind() == reflect.Struct {
		// Check for graphy tags on the struct itself
		// This requires finding the struct field in its parent if embedded
		// For now, we'll handle this during field processing when we detect
		// that this type is embedded

		// IMPORTANT: Store the type lookup BEFORE populating to handle circular references.
		// This prevents infinite recursion when types reference each other (A->B->A).
		// The type lookup will be incomplete at first, but that's OK - it will be
		// populated as we process the fields.
		g.typeLookups[typ] = result
		g.typeMutex.Unlock()

		// Now populate the type lookup. If this type is referenced circularly,
		// the recursive call will find it in the cache and return the partial result.
		g.populateTypeLookup(rootTyp, nil, result)
		return result
	}
	if typ == anyType {
		for _, at := range g.anyTypes {
			result.union[at.name] = at
			result.unionLowercase[strings.ToLower(at.name)] = at
		}
		// For each of the union types, add the fields to the result.
		result.mu.Lock()
		for _, at := range result.union {
			at.mu.RLock()
			for name, field := range at.fields {
				result.fields[name] = field
				result.fieldsLowercase[strings.ToLower(name)] = field
			}
			at.mu.RUnlock()
		}
		result.mu.Unlock()
		g.typeLookups[typ] = result
		g.typeMutex.Unlock()
		return result
	}
	// Fundamental types like floats and ints don't need these lookups because it doesn't make
	// sense in this context.
	result.fundamental = true
	g.typeLookups[typ] = result
	g.typeMutex.Unlock()
	return result
}

func (g *Graphy) dereferenceSlice(typ reflect.Type) (reflect.Type, *typeArrayModifier) {
	result := &typeArrayModifier{}
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		result.isPointer = true
	}
	for typ.Kind() == reflect.Slice {
		typ = typ.Elem()
		typ, result.array = g.dereferenceSlice(typ)
	}
	return typ, result
}

func (g *Graphy) getRequestStub(ctx context.Context, request string) (*RequestStub, error) {
	var timingContext *timing.Context
	var tCtx context.Context
	if g.EnableTiming {
		var complete timing.Complete
		timingContext, complete = timing.Start(ctx, "ParseRequest")
		defer complete()
		tCtx = timingContext
	} else {
		tCtx = ctx
	}

	if g.RequestCache == nil {
		if timingContext != nil {
			timingContext.AddDetails("cache", "none")
		}
		return g.newRequestStub(request)
	}

	stub, err := g.RequestCache.GetRequestStub(tCtx, request)
	if stub != nil || err != nil {
		if timingContext != nil {
			timingContext.AddDetails("cache", "hit")
		}
		return stub, err
	}

	if timingContext != nil {
		timingContext.AddDetails("cache", "miss")
	}

	stub, err = g.newRequestStub(request)
	g.RequestCache.SetRequestStub(tCtx, request, stub, err)
	return stub, err
}
