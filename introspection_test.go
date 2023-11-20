package quickgraph

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

var fullIntrospectionQuery = `
   query IntrospectionQuery {
     __schema {

       queryType { name }
       mutationType { name }
       subscriptionType { name }
       types {
         ...FullType
       }
       directives {
         name
         description

         locations
         args {
           ...InputValue
         }
       }
     }
   }

   fragment FullType on __Type {
     kind
     name
     description

     fields(includeDeprecated: true) {
       name
       description
       args {
         ...InputValue
       }
       type {
         ...TypeRef
       }
       isDeprecated
       deprecationReason
     }
     inputFields {
       ...InputValue
     }
     interfaces {
       ...TypeRef
     }
     enumValues(includeDeprecated: true) {
       name
       description
       isDeprecated
       deprecationReason
     }
     possibleTypes {
       ...TypeRef
     }
   }

   fragment InputValue on __InputValue {
     name
     description
     type { ...TypeRef }
     defaultValue
   }

   fragment TypeRef on __Type {
     kind
     name
     ofType {
       kind
       name
       ofType {
         kind
         name
         ofType {
           kind
           name
           ofType {
             kind
             name
             ofType {
               kind
               name
               ofType {
                 kind
                 name
                 ofType {
                   kind
                   name
                 }
               }
             }
           }
         }
       }
     }
   }
`

func TestGraphy_Introspection_Schema(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	g.RegisterFunction(ctx, FunctionDefinition{
		Name: "search",
		Function: func(search string) []SearchResultUnion {
			return []SearchResultUnion{
				{
					Human: &Human{},
				},
			}
		},
		Mode:           ModeQuery,
		ParameterNames: []string{"search"},
	})
	g.EnableIntrospection(ctx)

	// This query is from the RapidAPI app.

	result, err := g.ProcessRequest(ctx, fullIntrospectionQuery, "")
	assert.NoError(t, err)

	expected := `{
  "data": {
    "__schema": {
      "directives": [],
      "mutationType": {
        "name": "__mutation"
      },
      "queryType": {
        "name": "__query"
      },
      "subscriptionType": null,
      "types": [
        {
          "description": null,
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "SCALAR",
          "name": "",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [
            {
              "args": [
                {
                  "defaultValue": null,
                  "description": null,
                  "name": "arg1",
                  "type": {
                    "kind": "NON_NULL",
                    "name": "required",
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "int",
                      "ofType": null
                    }
                  }
                }
              ],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "FriendsConnection",
              "type": {
                "kind": "OBJECT",
                "name": "FriendsConnection",
                "ofType": null
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "appearsIn",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "LIST",
                  "name": "list",
                  "ofType": {
                    "kind": "NON_NULL",
                    "name": "required",
                    "ofType": {
                      "kind": "ENUM",
                      "name": "episode",
                      "ofType": null
                    }
                  }
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "friends",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "LIST",
                  "name": "list",
                  "ofType": {
                    "kind": "OBJECT",
                    "name": "Character",
                    "ofType": null
                  }
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "id",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "SCALAR",
                  "name": "string",
                  "ofType": null
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "name",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "SCALAR",
                  "name": "string",
                  "ofType": null
                }
              }
            }
          ],
          "inputFields": [],
          "interfaces": [],
          "kind": "OBJECT",
          "name": "Character",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "node",
              "type": {
                "kind": "OBJECT",
                "name": "Character",
                "ofType": null
              }
            }
          ],
          "inputFields": [],
          "interfaces": [],
          "kind": "OBJECT",
          "name": "ConnectionEdge",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [
            {
              "args": [
                {
                  "defaultValue": null,
                  "description": null,
                  "name": "arg1",
                  "type": {
                    "kind": "NON_NULL",
                    "name": "required",
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "int",
                      "ofType": null
                    }
                  }
                }
              ],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "FriendsConnection",
              "type": {
                "kind": "OBJECT",
                "name": "FriendsConnection",
                "ofType": null
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "appearsIn",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "LIST",
                  "name": "list",
                  "ofType": {
                    "kind": "NON_NULL",
                    "name": "required",
                    "ofType": {
                      "kind": "ENUM",
                      "name": "episode",
                      "ofType": null
                    }
                  }
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "friends",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "LIST",
                  "name": "list",
                  "ofType": {
                    "kind": "OBJECT",
                    "name": "Character",
                    "ofType": null
                  }
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "id",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "SCALAR",
                  "name": "string",
                  "ofType": null
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "name",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "SCALAR",
                  "name": "string",
                  "ofType": null
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "primaryFunction",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "SCALAR",
                  "name": "string",
                  "ofType": null
                }
              }
            }
          ],
          "inputFields": [],
          "interfaces": [
            {
              "kind": "NON_NULL",
              "name": "required",
              "ofType": {
                "kind": "OBJECT",
                "name": "Character",
                "ofType": null
              }
            }
          ],
          "kind": "OBJECT",
          "name": "Droid",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "edges",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "LIST",
                  "name": "list",
                  "ofType": {
                    "kind": "OBJECT",
                    "name": "ConnectionEdge",
                    "ofType": null
                  }
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "totalCount",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "SCALAR",
                  "name": "int",
                  "ofType": null
                }
              }
            }
          ],
          "inputFields": [],
          "interfaces": [],
          "kind": "OBJECT",
          "name": "FriendsConnection",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [
            {
              "args": [
                {
                  "defaultValue": null,
                  "description": null,
                  "name": "arg1",
                  "type": {
                    "kind": "NON_NULL",
                    "name": "required",
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "int",
                      "ofType": null
                    }
                  }
                }
              ],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "FriendsConnection",
              "type": {
                "kind": "OBJECT",
                "name": "FriendsConnection",
                "ofType": null
              }
            },
            {
              "args": [
                {
                  "defaultValue": null,
                  "description": null,
                  "name": "arg1",
                  "type": {
                    "kind": "SCALAR",
                    "name": "",
                    "ofType": null
                  }
                }
              ],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "Height",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "SCALAR",
                  "name": "float64",
                  "ofType": null
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "HeightMeters",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "SCALAR",
                  "name": "float64",
                  "ofType": null
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "appearsIn",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "LIST",
                  "name": "list",
                  "ofType": {
                    "kind": "NON_NULL",
                    "name": "required",
                    "ofType": {
                      "kind": "ENUM",
                      "name": "episode",
                      "ofType": null
                    }
                  }
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "friends",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "LIST",
                  "name": "list",
                  "ofType": {
                    "kind": "OBJECT",
                    "name": "Character",
                    "ofType": null
                  }
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "id",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "SCALAR",
                  "name": "string",
                  "ofType": null
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "name",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "SCALAR",
                  "name": "string",
                  "ofType": null
                }
              }
            }
          ],
          "inputFields": [],
          "interfaces": [
            {
              "kind": "NON_NULL",
              "name": "required",
              "ofType": {
                "kind": "OBJECT",
                "name": "Character",
                "ofType": null
              }
            }
          ],
          "kind": "OBJECT",
          "name": "Human",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "UNION",
          "name": "SearchResult",
          "possibleTypes": [
            {
              "kind": "OBJECT",
              "name": "Droid",
              "ofType": null
            },
            {
              "kind": "OBJECT",
              "name": "Human",
              "ofType": null
            },
            {
              "kind": "OBJECT",
              "name": "Starship",
              "ofType": null
            }
          ]
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "id",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "SCALAR",
                  "name": "string",
                  "ofType": null
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "name",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "SCALAR",
                  "name": "string",
                  "ofType": null
                }
              }
            }
          ],
          "inputFields": [],
          "interfaces": [],
          "kind": "OBJECT",
          "name": "Starship",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [
            {
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "EMPIRE"
            },
            {
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "JEDI"
            },
            {
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "NEWHOPE"
            }
          ],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "ENUM",
          "name": "episode",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "SCALAR",
          "name": "float64",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "SCALAR",
          "name": "int",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "SCALAR",
          "name": "string",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [
            {
              "args": [
                {
                  "defaultValue": null,
                  "description": null,
                  "name": "search",
                  "type": {
                    "kind": "NON_NULL",
                    "name": "required",
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "string",
                      "ofType": null
                    }
                  }
                }
              ],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "search",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "LIST",
                  "name": "list",
                  "ofType": {
                    "kind": "NON_NULL",
                    "name": "required",
                    "ofType": {
                      "kind": "UNION",
                      "name": "SearchResult",
                      "ofType": null
                    }
                  }
                }
              }
            }
          ],
          "inputFields": [],
          "interfaces": [],
          "kind": "OBJECT",
          "name": "__query",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "OBJECT",
          "name": "__mutation",
          "possibleTypes": []
        }
      ]
    }
  }
}`

	buff := bytes.Buffer{}
	err = json.Indent(&buff, []byte(result), "", "  ")
	assert.NoError(t, err)

	formatted := buff.String()

	assert.Equal(t, expected, formatted)
}

func TestGraphy_Introspection_Type(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	g.RegisterFunction(ctx, FunctionDefinition{
		Name: "search",
		Function: func(search string) []SearchResultUnion {
			return []SearchResultUnion{
				{
					Human: &Human{},
				},
			}
		},
		Mode:           ModeQuery,
		ParameterNames: []string{"search"},
	})
	g.EnableIntrospection(ctx)

	// This query is from the RapidAPI app.
	query := `
   query IntrospectionQuery {
     __type(name: "Character") {
       ...FullType
     }
   }

   fragment FullType on __Type {
     kind
     name
     description

     fields(includeDeprecated: false) {
       name
     }
   }
`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)

	expected := `{
  "data": {
    "__type": {
      "description": null,
      "fields": [
        {
          "name": "FriendsConnection"
        },
        {
          "name": "appearsIn"
        },
        {
          "name": "friends"
        },
        {
          "name": "id"
        },
        {
          "name": "name"
        }
      ],
      "kind": "OBJECT",
      "name": "Character"
    }
  }
}`

	buff := bytes.Buffer{}
	err = json.Indent(&buff, []byte(result), "", "  ")
	assert.NoError(t, err)

	formatted := buff.String()

	assert.Equal(t, expected, formatted)
}

type enumWithDescription string

func (e enumWithDescription) EnumValues() []EnumValue {
	return []EnumValue{
		{Name: "ENUM1", Description: "This is the first enum."},
		{Name: "ENUM-HALF", Description: "This is a half enum?", IsDeprecated: true, DeprecationReason: "This is deprecated."},
		{Name: "ENUM2", Description: "This is the second enum."},
	}
}

func TestGraphy_Introspection_Deprecation(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	type TestType struct {
		// This field is deprecated.
		DeprecatedField string `graphy:"name=deprecatedField,deprecated=This field is deprecated."`
		AnEnum          enumWithDescription
	}

	g.RegisterFunction(ctx, FunctionDefinition{
		Name: "sample",
		Function: func(input string) []TestType {
			return []TestType{
				{
					DeprecatedField: input,
				},
			}
		},
		Mode:           ModeQuery,
		ParameterNames: []string{"input"},
	})
	g.EnableIntrospection(ctx)

	// This query is from the RapidAPI app.
	result, err := g.ProcessRequest(ctx, fullIntrospectionQuery, "")
	assert.NoError(t, err)

	expected := `{
  "data": {
    "__schema": {
      "directives": [],
      "mutationType": {
        "name": "__mutation"
      },
      "queryType": {
        "name": "__query"
      },
      "subscriptionType": null,
      "types": [
        {
          "description": null,
          "enumValues": [],
          "fields": [
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "AnEnum",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "ENUM",
                  "name": "enumWithDescription",
                  "ofType": null
                }
              }
            },
            {
              "args": [],
              "deprecationReason": "This field is deprecated.",
              "description": null,
              "isDeprecated": true,
              "name": "deprecatedField",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "SCALAR",
                  "name": "string",
                  "ofType": null
                }
              }
            }
          ],
          "inputFields": [],
          "interfaces": [],
          "kind": "OBJECT",
          "name": "TestType",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [
            {
              "deprecationReason": "This is deprecated.",
              "description": "This is a half enum?",
              "isDeprecated": true,
              "name": "ENUM-HALF"
            },
            {
              "deprecationReason": null,
              "description": "This is the first enum.",
              "isDeprecated": false,
              "name": "ENUM1"
            },
            {
              "deprecationReason": null,
              "description": "This is the second enum.",
              "isDeprecated": false,
              "name": "ENUM2"
            }
          ],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "ENUM",
          "name": "enumWithDescription",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "SCALAR",
          "name": "string",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [
            {
              "args": [
                {
                  "defaultValue": null,
                  "description": null,
                  "name": "input",
                  "type": {
                    "kind": "NON_NULL",
                    "name": "required",
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "string",
                      "ofType": null
                    }
                  }
                }
              ],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "sample",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "LIST",
                  "name": "list",
                  "ofType": {
                    "kind": "NON_NULL",
                    "name": "required",
                    "ofType": {
                      "kind": "OBJECT",
                      "name": "TestType",
                      "ofType": null
                    }
                  }
                }
              }
            }
          ],
          "inputFields": [],
          "interfaces": [],
          "kind": "OBJECT",
          "name": "__query",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "OBJECT",
          "name": "__mutation",
          "possibleTypes": []
        }
      ]
    }
  }
}`

	buff := bytes.Buffer{}
	err = json.Indent(&buff, []byte(result), "", "  ")
	assert.NoError(t, err)

	formatted := buff.String()

	assert.Equal(t, expected, formatted)
}

func TestGraphy_Introspection_Interface(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	g.RegisterFunction(ctx, FunctionDefinition{
		Name: "sample",
		Function: func(input string) any {
			return Droid{
				Character: Character{
					Name: input,
				},
				PrimaryFunction: "droiding",
			}
		},
		Mode:              ModeQuery,
		ParameterNames:    []string{"input"},
		ReturnAnyOverride: []any{Character{}},
	})
	g.RegisterTypes(ctx, Droid{}, Character{})
	g.EnableIntrospection(ctx)

	// This query is from the RapidAPI app.
	result, err := g.ProcessRequest(ctx, fullIntrospectionQuery, "")
	assert.NoError(t, err)

	expected := `{
  "data": {
    "__schema": {
      "directives": [],
      "mutationType": {
        "name": "__mutation"
      },
      "queryType": {
        "name": "__query"
      },
      "subscriptionType": null,
      "types": [
        {
          "description": null,
          "enumValues": [],
          "fields": [
            {
              "args": [
                {
                  "defaultValue": null,
                  "description": null,
                  "name": "arg1",
                  "type": {
                    "kind": "NON_NULL",
                    "name": "required",
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "int",
                      "ofType": null
                    }
                  }
                }
              ],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "FriendsConnection",
              "type": {
                "kind": "OBJECT",
                "name": "FriendsConnection",
                "ofType": null
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "appearsIn",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "LIST",
                  "name": "list",
                  "ofType": {
                    "kind": "NON_NULL",
                    "name": "required",
                    "ofType": {
                      "kind": "ENUM",
                      "name": "episode",
                      "ofType": null
                    }
                  }
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "friends",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "LIST",
                  "name": "list",
                  "ofType": {
                    "kind": "INTERFACE",
                    "name": "Character",
                    "ofType": null
                  }
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "id",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "SCALAR",
                  "name": "string",
                  "ofType": null
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "name",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "SCALAR",
                  "name": "string",
                  "ofType": null
                }
              }
            }
          ],
          "inputFields": [],
          "interfaces": [],
          "kind": "INTERFACE",
          "name": "Character",
          "possibleTypes": [
            {
              "kind": "OBJECT",
              "name": "Droid",
              "ofType": null
            }
          ]
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "node",
              "type": {
                "kind": "INTERFACE",
                "name": "Character",
                "ofType": null
              }
            }
          ],
          "inputFields": [],
          "interfaces": [],
          "kind": "OBJECT",
          "name": "ConnectionEdge",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [
            {
              "args": [
                {
                  "defaultValue": null,
                  "description": null,
                  "name": "arg1",
                  "type": {
                    "kind": "NON_NULL",
                    "name": "required",
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "int",
                      "ofType": null
                    }
                  }
                }
              ],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "FriendsConnection",
              "type": {
                "kind": "OBJECT",
                "name": "FriendsConnection",
                "ofType": null
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "appearsIn",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "LIST",
                  "name": "list",
                  "ofType": {
                    "kind": "NON_NULL",
                    "name": "required",
                    "ofType": {
                      "kind": "ENUM",
                      "name": "episode",
                      "ofType": null
                    }
                  }
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "friends",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "LIST",
                  "name": "list",
                  "ofType": {
                    "kind": "INTERFACE",
                    "name": "Character",
                    "ofType": null
                  }
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "id",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "SCALAR",
                  "name": "string",
                  "ofType": null
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "name",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "SCALAR",
                  "name": "string",
                  "ofType": null
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "primaryFunction",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "SCALAR",
                  "name": "string",
                  "ofType": null
                }
              }
            }
          ],
          "inputFields": [],
          "interfaces": [
            {
              "kind": "NON_NULL",
              "name": "required",
              "ofType": {
                "kind": "INTERFACE",
                "name": "Character",
                "ofType": null
              }
            }
          ],
          "kind": "OBJECT",
          "name": "Droid",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "edges",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "LIST",
                  "name": "list",
                  "ofType": {
                    "kind": "OBJECT",
                    "name": "ConnectionEdge",
                    "ofType": null
                  }
                }
              }
            },
            {
              "args": [],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "totalCount",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "SCALAR",
                  "name": "int",
                  "ofType": null
                }
              }
            }
          ],
          "inputFields": [],
          "interfaces": [],
          "kind": "OBJECT",
          "name": "FriendsConnection",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [
            {
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "EMPIRE"
            },
            {
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "JEDI"
            },
            {
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "NEWHOPE"
            }
          ],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "ENUM",
          "name": "episode",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "SCALAR",
          "name": "int",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "UNION",
          "name": "sampleResultUnion",
          "possibleTypes": [
            {
              "kind": "NON_NULL",
              "name": "required",
              "ofType": {
                "kind": "INTERFACE",
                "name": "Character",
                "ofType": null
              }
            }
          ]
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "SCALAR",
          "name": "string",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [
            {
              "args": [
                {
                  "defaultValue": null,
                  "description": null,
                  "name": "input",
                  "type": {
                    "kind": "NON_NULL",
                    "name": "required",
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "string",
                      "ofType": null
                    }
                  }
                }
              ],
              "deprecationReason": null,
              "description": null,
              "isDeprecated": false,
              "name": "sample",
              "type": {
                "kind": "NON_NULL",
                "name": "required",
                "ofType": {
                  "kind": "UNION",
                  "name": "sampleResultUnion",
                  "ofType": null
                }
              }
            }
          ],
          "inputFields": [],
          "interfaces": [],
          "kind": "OBJECT",
          "name": "__query",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "OBJECT",
          "name": "__mutation",
          "possibleTypes": []
        }
      ]
    }
  }
}`

	buff := bytes.Buffer{}
	err = json.Indent(&buff, []byte(result), "", "  ")
	assert.NoError(t, err)

	formatted := buff.String()

	assert.Equal(t, expected, formatted)
}
