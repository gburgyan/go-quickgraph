![Build status](https://github.com/gburgyan/go-quickgraph/actions/workflows/go.yml/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/gburgyan/go-quickgraph)](https://goreportcard.com/report/github.com/gburgyan/go-quickgraph) [![PkgGoDev](https://pkg.go.dev/badge/github.com/gburgyan/go-quickgraph)](https://pkg.go.dev/github.com/gburgyan/go-quickgraph)

# About

`go-quickgraph` presents a simple code-first library for creating a GraphQL service in Go. The intent is to have a simple way to create a GraphQL service without having to write a lot of boilerplate code. Since this is a code-first implementation, the library is also able to generate a GraphQL schema from the set-up GraphQL environment.

# Design Goals

* Minimal boilerplate code. Write Go functions and structs, and the library will take care of the rest.
* Ability to process all valid GraphQL queries. This includes queries with fragments, aliases, variables, and directives.
* Generation of a GraphQL schema from the set-up GraphQL environment.
* Be as idiomatic as possible. This means using Go's native types and idioms as much as possible. If you are familiar with Go, you should be able to use this library without having to learn a lot of new concepts.
* If we need to relax something in the processing of a GraphQL query, err on the side of preserving Go idioms, even if it means that we are not 100% compliant with the GraphQL specification.
* Attempt to allow anything legal in GraphQL to be expressed in Go. Make the common things as easy as possible.
* Be as fast as practical. Cache aggressively.

The last point is critical in thinking about how the library is built. It relies heavily on Go reflection to figure out what the interfaces are of the functions and structs. Certain things, like function parameter names, are not available at runtime, so we have to make some compromises. If, for instance, we have a function that takes two non-context arguments, we can process requests assuming that the order of the parameters in the request matches the order from the function. This is not 100% compliant with the GraphQL specification, but it is a reasonable compromise to make in order to preserve Go idioms and minimize boilerplate code.

# Installation

```bash
go get github.com/gburgyan/go-quickgraph
```

# Usage and Examples

The examples here, as well as many of the unit tests are based directly on the examples from the GraphQL [documentation examples](https://graphql.org/learn/queries/).

An example service that uses this can be found in the https://github.com/gburgyan/go-quickgraph-sample repo.

```go
type Character struct {
	Id        string       `json:"id"`
	Name      string       `json:"name"`
	Friends   []*Character `json:"friends"`
	AppearsIn []Episode    `json:"appearsIn"`
}

type Episode string

// Go doesn't have a way reflecting these values, so we need to implement
// the EnumValues interface if we want to use them in our GraphQL schema.
// If not provided, the library will use the string representation of the
// enum values.
func (e Episode) EnumValues() []string {
    return []string{
        "NEWHOPE",
        "EMPIRE",
        "JEDI",
    }
}

func HeroProvider(ctx context.Context) Character {
	// Logic for resolving the Hero query
}

func RunExampleQuery(ctx context.Context) (string, err){
    g := Graphy{}
    g.RegisterProcessorWithParamNames(ctx, "hero", HeroProvider)
    input := `
    {
      hero {
        name
      }
    }`
    return g.ProcessRequest(ctx, input, "") // The last parameter is the variable JSON (optional)
}
```

## What's going on here?

When we define the `Character` type, it's simply a Go struct. We have decorated the struct's fields with the standard `json` tags that will be used to serialize the result to JSON. If those tags are omitted, the library will use the field names as the JSON keys.

There is also an enumeration for the `Episode`. Go doesn't natively support enums, so we have made a type that implements the `StringEnumValues` interface. This is used to generate the GraphQL schema as well as for soe validation during the processing of the request. This, as with the `json` tags, is also optional. The fallback is to just treat things as strings and leave the validation to the functions that process the request.

The `HeroProvider` function is the function that will be called when the `hero` query is processed. The function takes a context as a parameter, and returns a `Character` struct. This is the primary example of the "code-first" approach. We write the Go functions and structs, and the library takes care of the rest. There are ways to tweak the behavior of things, but the defaults should be sufficient for many use cases.

Finally, we set up a `Graphy` object and tell it about the function that will process the `hero` query. We then call the `ProcessRequest` function with the GraphQL query and the variable JSON (if any). The result is a JSON string that can be returned to the client.

In normal usage, we would initialize the `Graphy` object once, and then use it to process multiple requests. The `Graphy` object is thread-safe, so it can be used concurrently. It also caches all the reflection information that instructs it how to process the requests, so it is more efficient to reuse the same object. Additionally, it caches the parsed *queries* as well so if the same query is processed multiple times, it will be faster. This allows for the same query to be resused with different variable values.

# Theory of Operation

## Initialization of the Graphy Object

As the above example illustrates, the first step is to create a `Graphy` object. The intent is that an instance of this object is set up at service initialization time, and then used to process requests. The `Graphy` object is thread-safe, so it can be used concurrently.

The process is to add all the functions that can be used to service queries and mutations. The library will use reflection to figure out what the function signatures are, and will use that information to process the requests. The library will also use reflection to figure out what the types are of the parameters and return values, and will use that information to process the requests and generate the GraphQL schema.

Functions are simple Go `func`s that handle the processing. There are two broad "flavors" of functions: struct-based and anonymous. Since Go doesn't permit reflection of the parameters of functions we a few options. If the function takes one non-`Context` `struct` parameter, it will be treated as struct-based. The fields of the `struct` will be used to figure out the function signature. Otherwise, the function is an anonymous function. During the addition of a function, the ordered listing of parameter names can be passed. This will be covered in more detail further in this `README`.

Once all the functions are added, the `Graphy` object is ready to be used.

## Processing of a Request

Internally, a request is processed in four primary phases:

### Query Processing:

The inbound query is parsed into a `RequestStub` object. Since a request can have variables, the determination of the variable types can be done once, then cached. This is important because in complex cases, this can be an expensive operation as both the input and potential outputs must be considered.

### Variable Processing

If there are variables to be processed, they are unmarshalled using JSON. The result objects are then mapped into the variables that have been defined by the preceding step. Variables can represent both simple scalar types and more complex object graphs.

### Function Execution

Once the variables have been defined, the query or mutator functions can be called. This will call a function that was initially registered with the `Graphy` instance. 

### Output Generation

The most complex part of the overall request processing is the generation of the result object graph. All aspects of the standard GraphQL query language are supported. See the section on the [type systems](#type-system) for more information about how this operates.

Another aspect that GraphQL supports is operations that function on the returned data. This library models that as receiver functions on results variables:

```go

func (c *Character) FriendsConnection(first int) *FriendsConnection {
	// Processing goes here...
    return result
}

type FriendsConnection struct {
    TotalCount int               `json:"totalCount"`
    Edges      []*ConnectionEdge `json:"edges"`
}
```

This, in addition to the earlier example, will expose a function on the result object that is represented by a schema such as:

```graphql
type Character {
	appearsIn: [episode!]!
	friends: [Character]!
	FriendsConnection(arg1: Int!): FriendsConnection
	id: String!
	name: String!
}
```

There are several important things here:

* All public fields of the `Character` struct are reflected in the schema.
* The `json` tags are respected for the field naming
* The `FriendsConnection` receiver function is an [anonymous function](#anonymous-functions).

There is a further discussion on [schemata](#schema-generation) later on.

## Error handling

There are two general places where errors can occur: setup and runtime. During setup, the library will generally `panic` as this is something that should fail fast and indicates a structural problem with the program itself. At runtime, there should be no way that the system can `panic`.

When calling `result, err := g.ProcessRequest(ctx, input, "{variables...})` the result should contain something that can be returned to a GraphQL client in all cases. The errors, if any, are formatted the way that a GraphQL client would expect. So, for instance:

```json
{
  "data": {},
  "errors": [
    {
      "message": "error getting call parameters for function hero: invalid enum value INVALID",
      "locations": [
        {
          "line": 3,
          "column": 8
        }
      ],
      "path": [
        "hero"
      ]
    }
  ]
}
```

The error is also returned by the function itself and should be able to be handled normally.

# Functions

Functions are used in two ways in the processing of a `Graphy` request:

* Handling the primary query processing
* Providing processing while rendering the results from the output from the primary function

Any time a request is processed, it gets mapped to the function that is registered for that name. This is done with one of the `Register*` functions on the `Graphy` object.

In an effort to make it simple for developers to create these functions, there are several ways to add functions and `Graphy` that each have some advantages and disadvantages.

In all cases, if there is a `context.Context` parameter, that will be passed into the function.

## Anonymous Functions

Anonymous functions look like typical Golang functions and simply take a normal parameter list.

Given this simple function:

```go
func GetCourses(ctx context.Context, categories []*string) []*Course
```

You can register the processor with:

```go
g := Graphy{}
g.RegisterProcessor(ctx, "courses", GetCourses)
```

The downside of this approach is based on a limitation in reflection in Golang: there is no way to get the _names_ of the parameters. In this case `Graphy` will run assuming the parameters are positional. This is a variance from orthodox GraphQL behavior as it expects named parameters. If a schema is generated from the `Graphy`, these will be rendered as `arg1`, `arg2`, etc. Those are purely placeholders for the schema and the processing of the request will still be treated positionally; the names of the passed parameters are ignored.

If the named parameters are needed, you can do that via:

```go
g := Graphy{}
g.RegisterProcessorWithParamNames(ctx, "courses", GetCourses, "categories")
```

Once you tell `Graphy` the names of the paramters to expect, it can function with named parameters as is typical for GraphQL.

Finally, you can also register the processor with the most flexible function:

```go
g := Graphy{}
g.RegisterFunction(ctx, graphy.FunctionDefinition{
	Name: "courses",
	Function: GetCourses,
	ParameterNames: {"categories"},
	Mode: graphy.ModeQuery,
})
```

This is also how you can register a mutation instead of the default query.

A function may also take zero non-`Context` parameters, in which case it simply gets run.

## Struct Functions

The alternate way that a function can be defined is by passing in a struct as the parameter. The structure must be passed as value. In addition to the struct parameter, it may also, optionally, have a `context.Context` parameter.

For example:

```go
type CourseInput struct {
	Categories []string
}

func GetCourses(ctx context.Context, in CourseInput) []*Course
{
	// Implementation
}

g := Graphy{}
g.RegisterProcessor(ctx, "courses", GetCourses)
```

Since reflection can be used to get the names of the members of the structure, that information will be used to get the names of the parameters that will be exposed for the function.

The same `RegisterFunction` function can also be used as described above to define additional aspects of the function, such as if it's a mutator.

## Return Values

Regardless of how the function is defined, it is required to return a struct, a pointer to a struct, or a slice of either. It may optionally return an `error` as well. The returned value will be used to populate the response to the GraphQL calls. The shape of the response object will be used to construct the schema of the `Graphy` in case that is used.

There is a special case where a function can return an `any` type. This is valid from a runtime perspective as the type of the object can be determined at runtime, but it precludes schema generation for the result as the type of the result cannot be determined by the signature of the function.

## Output Functions

When calling a function to service a request, that function returns the value that is processed into the response -- that part is obvious. Another feature is that those objects can have functions on them as well. This plays into the overall Graph functionality that is exposed by `Graphy`. These receiver functions follow the same pattern as above.

Additionally, they get transformed into schemas exactly as expected. If a receiver function takes nothing (or only a `context.Context` object), then it gets exposed as a field. If the field is referenced, then the function is invoked and the output generation continues. If the function takes parameters, then it's exposed as a function with parameters both for the request as well as the schema:

```go
type Human struct {
	Character
	HeightMeters float64 `json:"HeightMeters"`
}

func (h *Human) Height(units *string) float64 {
	if units == nil {
		return h.HeightMeters
	}
	if *units == "FOOT" {
		return roundToPrecision(h.HeightMeters*3.28084, 7)
	}
	return h.HeightMeters
}
```

Since the `Height` function takes a pointer to a `string` as a parameter, it's treated as optional.

In this case both of the following queries will work:

```graphql
{
  Human(id: "1000") {
    name
    height
  }
}
```

```graphql
{
    Human(id: "1000") {
        name
        height(unit: FOOT)
    }
}
```

## Function Parameters

Regardless of how the function is invoked, the parameters for the function come from either the base query itself or variables that are passed in along with the query. `Graphy` supports both scalar types, as well as more complex types including complex, and even nested, structures, as well as slices of those objects.

# Type System

The way `Graphy` works with types is intended to be as transparent to the user as possible. The normal types, scalars, structs, and slices all work as expected. This applies to both input in the form of parameters being sent in to functions and the results of those functions.

Maps, presently, are not supported.

## Enums

Go doesn't have a native way of representing enumerations in a way that is open to be used for reflection. To get around this, `Graphy` provides a few different ways of exposing enumerations.

### Simple strings

### `EnumUnmarshaler` interface

If you need to have function inputs that map to specific non-string inputs, you can implement the `EnumUnmarshaler` interface:

```go
// EnumUnmarshaler provides an interface for types that can unmarshal
// a string representation into their enumerated type. This is useful
// for types that need to convert a string, typically from external sources
// like JSON or XML, into a specific enumerated type in Go.
//
// UnmarshalString should return the appropriate enumerated value for the
// given input string, or an error if the input is not valid for the enumeration.
type EnumUnmarshaler interface {
	UnmarshalString(input string) (interface{}, error)
}
```

`UnmarshalString` is called with the supplied identifier, and is responsible for converting that into whatever type is needed. If the identifier cannot be converted, simply return an error.

The downside of this is that there is no way to communicate to the schema what are the _valid_ values for the enumeration.

Example:

```go
type MyEnum string

const (
	EnumVal1 MyEnum = "EnumVal1"
	EnumVal2 MyEnum = "EnumVal2"
	EnumVal3 MyEnum = "EnumVal3"
)

func (e *MyEnum) UnmarshalString(input string) (interface{}, error) {
	switch input {
	case "EnumVal1":
		return EnumVal1, nil
	case "EnumVal2":
		return EnumVal2, nil
	case "EnumVal3":
		return EnumVal3, nil
	default:
		return nil, fmt.Errorf("invalid enum value %s", input)
	}
}
```

In this case, the enum type is a string, but that's not a requirement.

### `StringEnumValues`

Another way of dealing with enumerations is to treat them as strings, but with a layer of validation applied. You can implement the `StringEnumValues` interface to say what are the valid values for a given type.

```go
// StringEnumValues provides an interface for types that can return
// a list of valid string representations for their enumeration.
// This can be useful in scenarios like validation or auto-generation
// of documentation where a list of valid enum values is required.
//
// EnumValues should return a slice of strings representing the valid values
// for the enumeration.
type StringEnumValues interface {
	EnumValues() []string
}
```

These strings are used both for input validation and schema generation. The limitation is that the inputs and outputs that use this type need to be of a string type.

An example from the tests:

```go
type episode string

func (e episode) EnumValues() []string {
	return []string{
		"NEWHOPE",
		"EMPIRE",
		"JEDI",
	}
}
```

## Interfaces

Interfaces, in this case, are referring to how GraphQL uses the term "interface." The way that a type can implement an interface, as well as select the output filtering based on the type of object that is being returned.

The way this is modeled in this library is by using anonymous fields on a struct type to show an "is-a" relationship.

So, for instance:

```go
type Character struct {
	Id        string       `json:"id"`
	Name      string       `json:"name"`
	Friends   []*Character `json:"friends"`
	AppearsIn []episode    `json:"appearsIn"`
}

type Human struct {
	Character
	HeightMeters float64 `json:"HeightMeters"`
}
```

In this case, a `Human` is a subtype of `Character`. The schema generated from this is:

```graphql
type Human implements Character {
	FriendsConnection(arg1: Int!): FriendsConnection
	HeightMeters: Float!
}

type Character {
	appearsIn: [episode!]!
	friends: [Character]!
	id: String!
	name: String!
}
```

## Unions

Another aspect of GraphQL that doesn't cleanly map to Go is the concept of unions -- where a value can be one of several distinct types.

This is handled by one of two ways: implicit and explicit unions.

### Implicit Unions

Implicit unions are created by functions that return multiple pointers to results. Of course only one of those result pointers can be non-nil. For example:

```go
type resultA struct {
	OutStringA string
}
type resultB struct {
	OutStringB string
}
func Implicit(ctx context.Context, selector string) (*resultA, *resultB, error) {
	// implementation
}
```

This will generate a schema that looks like:

```graphql
type Query {
	Implicit(arg1: String!): ImplicitResultUnion!
}

union ImplicitResultUnion = resultA | resultB

type resultA {
	OutStringA: String!
}

type resultB {
	OutStringB: String!
}
```

If you need a custom-named enum, you can register the function like:

```go
g.RegisterFunction(ctx, FunctionDefinition{
	Name:            "CustomResultFunc",
	Function:        function,
	ReturnUnionName: "MyUnion",
})
```

In which case the name of the union is `MyUnion`.

### Explicit Unions

You can also name a type ending with the string `Union` and that type will be treated as a union. The members of that type must all be pointers. The result of the evaluation of the union must have a single non-nil value, and that is the implied type of the result.

# Schema Generation

Once a `graphy` is set up with all the query and mutation handlers, you can call:

```go
schema, err := g.SchemaDefinition(ctx)
```

This will create a GraphQL schema that represents the state of the `graphy` object. Explore the `schema_type_test.go` test file for more examples of generated schemata.

## Limitations

* Presently the input and output objects are treated equivalently
* If there are multiple types with the same name, but from different packages, the results will not be valid.
 
# Caching

Caching is an optional feature of the graph processing. To enable it, simply set the `RequestCache` on the `Graphy` object. The cache is an implementation of the `GraphRequestCache` interface. If this is not set, the graphy functionality will not cache anything.

The cache is used to cache the result of parsing the request. This is a `RequestStub` as well as any errors that were present in parsing the errors. The request stub contains everything that was prepared to run the request except the variables that were passed in. This process involves a lot of reflection, so this is a comparatively expensive operation. By caching this processing, we gain a roughly 10x speedup.

We cache errors as well because a request that can't be fulfilled by the `Graphy` library will continue to be an error even if it submitted again -- there is no reason to reprocess the request to simply get back to the answer of error.

The internals of the `RequestStub` is only in-memory and not externally serializable.

## Example implementation

Using a simple cache library `github.com/patrickmn/go-cache`, here's a simple implementation:

```go
type SimpleGraphRequestCache struct {
	cache *cache.Cache
}

type simpleGraphRequestCacheEntry struct {
	request string
	stub    *RequestStub
	err     error
}

func (d *SimpleGraphRequestCache) SetRequestStub(ctx context.Context, request string, stub *RequestStub, err error) {
	setErr := d.cache.Add(request, &simpleGraphRequestCacheEntry{
		request: request,
		stub:    stub,
		err:     err,
	}, time.Hour)
	if setErr != nil {
		// Log this error, but don't return it.
		// Potentially disable the cache if this recurs continuously.
	}
}

func (d *SimpleGraphRequestCache) GetRequestStub(ctx context.Context, request string) (*RequestStub, error) {
	value, found := d.cache.Get(request)
	if !found {
		return nil, nil
	}
	entry, ok := value.(*simpleGraphRequestCacheEntry)
	if !ok {
		return nil, nil
	}
	return entry.stub, entry.err
}
```

Since each unique request, independent of the variables, can be cached, it's important to have a working eviction policy to prevent a denial of service attack from exhausting memory.

## Internal caching

Internally `Graphy` will cache much of the results of reflection operations. These relate to the types that are used for input and output. Since these have a one-to-one relationship to the internal types of the running system, they are cached by `Graphy` for the lifetime of the object; it can't grow out of bounds and cannot be subject to a denial of service attack. 

# Dealing with unknown commands

A frequent requirement is to implement a strangler pattern to start taking requests for things that can be processed, but to forward requests that can't be processed to another service. This is enabled by the processing pipeline by returning a `UnknownCommandError`. Since the processing of the request can be cached, this can be a fail-fast scenario so that the request could be forwarded to another service for processing. 

# Benchmarks

Given this relatively complex query:

```graphql
mutation CreateReviewForEpisode($ep: Episode!, $review: ReviewInput!) {
  createReview(episode: $ep, review: $review) {
    stars
    commentary
  }
}
```

with this variable JSON:

```json
{
  "ep": "JEDI",
  "review": {
    "stars": 5,
    "commentary": "This is a great movie!"
  }
}
```

with caching enabled, the framework overhead is less than 4.8µs on an Apple M1 Pro processor. This includes parsing the variable JSON, calling the function for `CreateReviewForEpisode`, and processing the output. The vast majority of the overhead, roughly 75% of the time, isn't the library itself, but rather the unmarshalling of the variable JSON as well as marshaling the result to be returned.

See the `benchmark_test.go` benchmarks for more tests and to evaluate this on your hardware.

While potentially caching the variable JSON would be possible, the decision was made that it's likely not worthwhile as the variables are what are most likely to change between requests negating any benefits of caching.

# General Limitations

## Validation of Input

