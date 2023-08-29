![Build status](https://github.com/gburgyan/go-quickgraph/actions/workflows/go.yml/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/gburgyan/go-quickgraph)](https://goreportcard.com/report/github.com/gburgyan/go-quickgraph) [![PkgGoDev](https://pkg.go.dev/badge/github.com/gburgyan/go-quickgraph)](https://pkg.go.dev/github.com/gburgyan/go-quickgraph)

# About

`go-quickgraph` presents a simple code-first library for creating a GraphQL service in Go. The intent is to have a simple way to create a GraphQL service without having to write a lot of boilerplate code. Since this is a code-first implementation, the library is also able to generate a GraphQL schema from the set-up GraphQL environment.

# Design Goals

* Minimal boilerplate code. Write Go functions and structs, and the library will take care of the rest.
* Ability to process all valid GraphQL queries. This includes queries with fragments, aliases, variables, and directives.
* Generation of a GraphQL schema from the set-up GraphQL environment.
* Be as idiomatic as possible. This means using Go's native types and idioms as much as possible. If you are familiar with Go, you should be able to use this library without having to learn a lot of new concepts.
* If we need to relax something in the processing of a GraphQL query, err on the side of preserving Go idioms, even if it means that we are not 100% compliant with the GraphQL specification.
* Be as fast as practical. Cache aggressively.

The last point is critical in thinking about how the library is built. It relies heavily on Go reflection to figure out what the interfaces are of the functions and structs. Certain things, like function parameter names, are not available at runtime, so we have to make some compromises. If, for instance, we have a function that takes two non-context arguments, we can process requests assuming that the order of the parameters in the request matches the order from the function. This is not 100% compliant with the GraphQL specification, but it is a reasonable compromise to make in order to preserve Go idioms and minimize boilerplate code.

# Installation

# Usage and Examples

The examples here, as well as many of the unit tests are based directly on the examples from the GraphQL [documentation examples](https://graphql.org/learn/queries/).

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

# Type System

## Enums

## Interfaces

## Unions

# Functions

## Struct Functions vs. Anonymous Functions

## Helper Functions

# Limitations


