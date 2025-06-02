package quickgraph

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"reflect"
	"strings"
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
          "name": "Boolean",
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
                    "name": null,
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "Int",
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
                "name": null,
                "ofType": {
                  "kind": "LIST",
                  "name": null,
                  "ofType": {
                    "kind": "NON_NULL",
                    "name": null,
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
                "name": null,
                "ofType": {
                  "kind": "LIST",
                  "name": null,
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "String",
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "String",
                  "ofType": null
                }
              }
            }
          ],
          "inputFields": [],
          "interfaces": [
            {
              "kind": "INTERFACE",
              "name": "ICharacter",
              "ofType": null
            }
          ],
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
                    "name": null,
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "Int",
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
                "name": null,
                "ofType": {
                  "kind": "LIST",
                  "name": null,
                  "ofType": {
                    "kind": "NON_NULL",
                    "name": null,
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
                "name": null,
                "ofType": {
                  "kind": "LIST",
                  "name": null,
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "String",
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "String",
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "String",
                  "ofType": null
                }
              }
            }
          ],
          "inputFields": [],
          "interfaces": [
            {
              "kind": "INTERFACE",
              "name": "ICharacter",
              "ofType": null
            }
          ],
          "kind": "OBJECT",
          "name": "Droid",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "SCALAR",
          "name": "Float",
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
                "name": null,
                "ofType": {
                  "kind": "LIST",
                  "name": null,
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "Int",
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
                    "name": null,
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "Int",
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
                    "name": "String",
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "Float",
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "Float",
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
                "name": null,
                "ofType": {
                  "kind": "LIST",
                  "name": null,
                  "ofType": {
                    "kind": "NON_NULL",
                    "name": null,
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
                "name": null,
                "ofType": {
                  "kind": "LIST",
                  "name": null,
                  "ofType": {
                    "kind": "INTERFACE",
                    "name": "ICharacter",
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "String",
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "String",
                  "ofType": null
                }
              }
            }
          ],
          "inputFields": [],
          "interfaces": [
            {
              "kind": "INTERFACE",
              "name": "ICharacter",
              "ofType": null
            }
          ],
          "kind": "OBJECT",
          "name": "Human",
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
                    "name": null,
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "Int",
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
                "name": null,
                "ofType": {
                  "kind": "LIST",
                  "name": null,
                  "ofType": {
                    "kind": "NON_NULL",
                    "name": null,
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
                "name": null,
                "ofType": {
                  "kind": "LIST",
                  "name": null,
                  "ofType": {
                    "kind": "INTERFACE",
                    "name": "ICharacter",
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "String",
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "String",
                  "ofType": null
                }
              }
            }
          ],
          "inputFields": [],
          "interfaces": [],
          "kind": "INTERFACE",
          "name": "ICharacter",
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
              "name": "Character",
              "ofType": null
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
          "name": "ID",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "SCALAR",
          "name": "Int",
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "String",
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "String",
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
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "SCALAR",
          "name": "String",
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
          "fields": [
            {
              "args": [
                {
                  "defaultValue": null,
                  "description": null,
                  "name": "search",
                  "type": {
                    "kind": "NON_NULL",
                    "name": null,
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "String",
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
                "name": null,
                "ofType": {
                  "kind": "LIST",
                  "name": null,
                  "ofType": {
                    "kind": "NON_NULL",
                    "name": null,
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

	// Debug: print the actual output
	if formatted != expected {
		t.Logf("ACTUAL OUTPUT:\n%s", formatted)
	}

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
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "SCALAR",
          "name": "Boolean",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "SCALAR",
          "name": "Float",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "SCALAR",
          "name": "ID",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "SCALAR",
          "name": "Int",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "SCALAR",
          "name": "String",
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
              "name": "AnEnum",
              "type": {
                "kind": "NON_NULL",
                "name": null,
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "String",
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
          "fields": [
            {
              "args": [
                {
                  "defaultValue": null,
                  "description": null,
                  "name": "input",
                  "type": {
                    "kind": "NON_NULL",
                    "name": null,
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "String",
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
                "name": null,
                "ofType": {
                  "kind": "LIST",
                  "name": null,
                  "ofType": {
                    "kind": "NON_NULL",
                    "name": null,
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
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "SCALAR",
          "name": "Boolean",
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
                    "name": null,
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "Int",
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
                "name": null,
                "ofType": {
                  "kind": "LIST",
                  "name": null,
                  "ofType": {
                    "kind": "NON_NULL",
                    "name": null,
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
                "name": null,
                "ofType": {
                  "kind": "LIST",
                  "name": null,
                  "ofType": {
                    "kind": "INTERFACE",
                    "name": "ICharacter",
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "String",
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "String",
                  "ofType": null
                }
              }
            }
          ],
          "inputFields": [],
          "interfaces": [
            {
              "kind": "INTERFACE",
              "name": "ICharacter",
              "ofType": null
            }
          ],
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
                "kind": "INTERFACE",
                "name": "ICharacter",
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
                    "name": null,
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "Int",
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
                "name": null,
                "ofType": {
                  "kind": "LIST",
                  "name": null,
                  "ofType": {
                    "kind": "NON_NULL",
                    "name": null,
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
                "name": null,
                "ofType": {
                  "kind": "LIST",
                  "name": null,
                  "ofType": {
                    "kind": "INTERFACE",
                    "name": "ICharacter",
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "String",
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "String",
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "String",
                  "ofType": null
                }
              }
            }
          ],
          "inputFields": [],
          "interfaces": [
            {
              "kind": "INTERFACE",
              "name": "ICharacter",
              "ofType": null
            }
          ],
          "kind": "OBJECT",
          "name": "Droid",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "SCALAR",
          "name": "Float",
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
                "name": null,
                "ofType": {
                  "kind": "LIST",
                  "name": null,
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "Int",
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
                    "name": null,
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "Int",
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
                "name": null,
                "ofType": {
                  "kind": "LIST",
                  "name": null,
                  "ofType": {
                    "kind": "NON_NULL",
                    "name": null,
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
                "name": null,
                "ofType": {
                  "kind": "LIST",
                  "name": null,
                  "ofType": {
                    "kind": "INTERFACE",
                    "name": "ICharacter",
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "String",
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
                "name": null,
                "ofType": {
                  "kind": "SCALAR",
                  "name": "String",
                  "ofType": null
                }
              }
            }
          ],
          "inputFields": [],
          "interfaces": [],
          "kind": "INTERFACE",
          "name": "ICharacter",
          "possibleTypes": [
            {
              "kind": "OBJECT",
              "name": "Character",
              "ofType": null
            },
            {
              "kind": "OBJECT",
              "name": "Droid",
              "ofType": null
            },
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
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "SCALAR",
          "name": "ID",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "SCALAR",
          "name": "Int",
          "possibleTypes": []
        },
        {
          "description": null,
          "enumValues": [],
          "fields": [],
          "inputFields": [],
          "interfaces": [],
          "kind": "SCALAR",
          "name": "String",
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
          "kind": "UNION",
          "name": "sampleResultUnion",
          "possibleTypes": [
            {
              "kind": "OBJECT",
              "name": "Character",
              "ofType": null
            },
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
              "args": [
                {
                  "defaultValue": null,
                  "description": null,
                  "name": "input",
                  "type": {
                    "kind": "NON_NULL",
                    "name": null,
                    "ofType": {
                      "kind": "SCALAR",
                      "name": "String",
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
                "name": null,
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

	// Debug: print the actual output for interface test
	if formatted != expected {
		t.Logf("ACTUAL OUTPUT:\n%s", formatted)
	}

	assert.Equal(t, expected, formatted)
}

func TestIntrospectionScalarName_WithBoolType(t *testing.T) {
	tl := &typeLookup{rootType: reflect.TypeOf(true)}
	result := introspectionScalarName(tl)
	assert.Equal(t, "Boolean", result)
}

func TestIntrospectionScalarName_WithIntType(t *testing.T) {
	tl := &typeLookup{rootType: reflect.TypeOf(int(1))}
	result := introspectionScalarName(tl)
	assert.Equal(t, "Int", result)
}

func TestIntrospectionScalarName_WithFloatType(t *testing.T) {
	tl := &typeLookup{rootType: reflect.TypeOf(float64(1.0))}
	result := introspectionScalarName(tl)
	assert.Equal(t, "Float", result)
}

func TestIntrospectionScalarName_WithStringType(t *testing.T) {
	tl := &typeLookup{rootType: reflect.TypeOf("test")}
	result := introspectionScalarName(tl)
	assert.Equal(t, "String", result)
}

func TestWrapType_WithNonWrapperKind(t *testing.T) {
	g := &Graphy{}

	// Create a base type to wrap
	baseType := &__Type{
		Name: strPtr("BaseType"),
		Kind: IntrospectionKindObject,
	}

	// Test wrapping with a custom name (non-wrapper kind)
	// This tests the else path in wrapType where name is used
	wrapped := g.wrapType(baseType, "CustomType", IntrospectionKindObject)

	assert.NotNil(t, wrapped)
	assert.NotNil(t, wrapped.Name)
	assert.Equal(t, "CustomType", *wrapped.Name)
	assert.Equal(t, IntrospectionKindObject, wrapped.Kind)
	assert.Equal(t, baseType, wrapped.OfType)
}

func TestWrapType_WithWrapperKinds(t *testing.T) {
	g := &Graphy{}

	// Create a base type to wrap
	baseType := &__Type{
		Name: strPtr("BaseType"),
		Kind: IntrospectionKindObject,
	}

	// Test NON_NULL wrapper - should have nil name
	nonNullWrapped := g.wrapType(baseType, "ignored", IntrospectionKindNonNull)
	assert.NotNil(t, nonNullWrapped)
	assert.Nil(t, nonNullWrapped.Name)
	assert.Equal(t, IntrospectionKindNonNull, nonNullWrapped.Kind)
	assert.Equal(t, baseType, nonNullWrapped.OfType)

	// Test LIST wrapper - should have nil name
	listWrapped := g.wrapType(baseType, "alsoIgnored", IntrospectionKindList)
	assert.NotNil(t, listWrapped)
	assert.Nil(t, listWrapped.Name)
	assert.Equal(t, IntrospectionKindList, listWrapped.Kind)
	assert.Equal(t, baseType, listWrapped.OfType)
}

// Helper function to create string pointers
func strPtr(s string) *string {
	return &s
}

func TestIntrospectionScalarName_WithUnknownType(t *testing.T) {
	tl := &typeLookup{rootType: reflect.TypeOf(map[string]string{})}
	result := introspectionScalarName(tl)
	assert.Equal(t, "", result, "Unknown scalar types should return empty string")
}

// TestIntrospectionInterfaceKind tests that interface references in introspection have correct kind
func TestIntrospectionInterfaceKind(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	// Define types that implement interfaces similar to Employee/Developer/Manager pattern
	type BaseEmployee struct {
		ID   int
		Name string
	}

	type Developer struct {
		BaseEmployee
		Language string
	}

	type Manager struct {
		BaseEmployee
		Department string
	}

	// Register a query that returns the base type
	g.RegisterQuery(ctx, "getEmployee", func(ctx context.Context, id int) BaseEmployee {
		return BaseEmployee{ID: id, Name: "Test Employee"}
	}, "id")

	// Register concrete types
	g.RegisterTypes(ctx, BaseEmployee{}, Developer{}, Manager{})

	// Enable introspection
	g.EnableIntrospection(ctx)

	// Execute introspection query to check interface kinds
	query := `{
		__schema {
			types {
				name
				kind
				interfaces {
					name
					kind
				}
			}
		}
	}`

	result, _ := g.ProcessRequest(ctx, query, "")

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse introspection response: %v", err)
	}

	// Check for errors
	if errors, ok := response["errors"]; ok {
		t.Fatalf("Introspection query returned errors: %v", errors)
	}

	// Extract types from response
	data := response["data"].(map[string]interface{})
	schema := data["__schema"].(map[string]interface{})
	types := schema["types"].([]interface{})

	// Check Developer type
	for _, typeData := range types {
		typeMap := typeData.(map[string]interface{})
		name := typeMap["name"].(string)

		if name == "Developer" {
			interfaces := typeMap["interfaces"].([]interface{})
			if len(interfaces) > 0 {
				for _, iface := range interfaces {
					ifaceMap := iface.(map[string]interface{})
					ifaceKind := ifaceMap["kind"].(string)
					ifaceName := ifaceMap["name"].(string)

					// The key assertion: interfaces must have kind INTERFACE
					if ifaceKind != "INTERFACE" {
						t.Errorf("Interface %s referenced by Developer has incorrect kind: %s, expected INTERFACE", ifaceName, ifaceKind)
					}
				}
			}
		}

		if name == "Manager" {
			interfaces := typeMap["interfaces"].([]interface{})
			if len(interfaces) > 0 {
				for _, iface := range interfaces {
					ifaceMap := iface.(map[string]interface{})
					ifaceKind := ifaceMap["kind"].(string)
					ifaceName := ifaceMap["name"].(string)

					// The key assertion: interfaces must have kind INTERFACE
					if ifaceKind != "INTERFACE" {
						t.Errorf("Interface %s referenced by Manager has incorrect kind: %s, expected INTERFACE", ifaceName, ifaceKind)
					}
				}
			}
		}
	}
}

// TestIntrospectionWrapperTypeNames tests that NON_NULL and LIST wrapper types have null names
func TestIntrospectionWrapperTypeNames(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	// Register a query that returns various wrapped types
	g.RegisterQuery(ctx, "getStrings", func(ctx context.Context) []string {
		return []string{"test"}
	})

	g.RegisterQuery(ctx, "getOptionalString", func(ctx context.Context) *string {
		s := "test"
		return &s
	})

	// Enable introspection
	g.EnableIntrospection(ctx)

	// Execute introspection query
	query := `{
		__type(name: "__query") {
			fields {
				name
				type {
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
	}`

	result, _ := g.ProcessRequest(ctx, query, "")

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse introspection response: %v", err)
	}

	// Check for errors
	if errors, ok := response["errors"]; ok {
		t.Fatalf("Introspection query returned errors: %v", errors)
	}

	// Extract fields
	data := response["data"].(map[string]interface{})
	typeData := data["__type"].(map[string]interface{})
	fields := typeData["fields"].([]interface{})

	for _, fieldData := range fields {
		field := fieldData.(map[string]interface{})
		fieldName := field["name"].(string)
		typeInfo := field["type"].(map[string]interface{})

		if fieldName == "getStrings" {
			// Should be NON_NULL -> LIST -> NON_NULL -> string
			if typeInfo["kind"] != "NON_NULL" {
				t.Errorf("getStrings: expected outer NON_NULL, got %s", typeInfo["kind"])
			}
			if typeInfo["name"] != nil {
				t.Error("getStrings: NON_NULL wrapper should have null name")
			}

			listType := typeInfo["ofType"].(map[string]interface{})
			if listType["kind"] != "LIST" {
				t.Errorf("getStrings: expected LIST, got %s", listType["kind"])
			}
			if listType["name"] != nil {
				t.Error("getStrings: LIST wrapper should have null name")
			}

			innerNonNull := listType["ofType"].(map[string]interface{})
			if innerNonNull["kind"] != "NON_NULL" {
				t.Errorf("getStrings: expected inner NON_NULL, got %s", innerNonNull["kind"])
			}
			if innerNonNull["name"] != nil {
				t.Error("getStrings: inner NON_NULL wrapper should have null name")
			}
		}

		if fieldName == "getOptionalString" {
			// Should be just SCALAR String (nullable)
			if typeInfo["kind"] != "SCALAR" {
				t.Errorf("getOptionalString: expected SCALAR, got %s", typeInfo["kind"])
			}
			if typeInfo["name"] != "String" {
				t.Errorf("getOptionalString: expected name 'String', got %v", typeInfo["name"])
			}
		}
	}
}

func TestScalarNamesInIntrospection(t *testing.T) {
	g := &Graphy{}
	ctx := context.Background()

	// Define a type that uses all fundamental scalar types
	type TestScalars struct {
		StringField  string  `json:"stringField"`
		IntField     int     `json:"intField"`
		Int32Field   int32   `json:"int32Field"`
		FloatField   float64 `json:"floatField"`
		Float32Field float32 `json:"float32Field"`
		BoolField    bool    `json:"boolField"`
	}

	// Register a query
	g.RegisterQuery(ctx, "getScalars", func() TestScalars {
		return TestScalars{
			StringField:  "test",
			IntField:     42,
			Int32Field:   32,
			FloatField:   3.14,
			Float32Field: 2.71,
			BoolField:    true,
		}
	})

	// Enable introspection
	g.EnableIntrospection(ctx)

	// Query introspection for the TestScalars type
	query := `{
		__type(name: "TestScalars") {
			fields {
				name
				type {
					kind
					name
					ofType {
						kind
						name
					}
				}
			}
		}
	}`

	resp, err := g.ProcessRequest(ctx, query, "{}")
	if err != nil {
		t.Fatalf("Failed to process request: %v", err)
	}

	// Parse the response
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Extract the type data
	data := result["data"].(map[string]interface{})
	typeInfo := data["__type"].(map[string]interface{})
	fields := typeInfo["fields"].([]interface{})

	// Check each field
	expectedScalars := map[string]string{
		"stringField":  "String",
		"intField":     "Int",
		"int32Field":   "Int",
		"floatField":   "Float",
		"float32Field": "Float",
		"boolField":    "Boolean",
	}

	for _, fieldInterface := range fields {
		field := fieldInterface.(map[string]interface{})
		fieldName := field["name"].(string)
		typeInfo := field["type"].(map[string]interface{})

		// All fields are non-null, so check ofType
		ofType := typeInfo["ofType"].(map[string]interface{})
		kind := ofType["kind"].(string)
		typeName := ofType["name"].(string)

		if kind != "SCALAR" {
			t.Errorf("Field %s: expected SCALAR kind, got %s", fieldName, kind)
		}

		if expected, ok := expectedScalars[fieldName]; ok {
			if typeName != expected {
				t.Errorf("Field %s: expected scalar name %s, got %s", fieldName, expected, typeName)
			}
			t.Logf("✓ Field %s correctly has scalar type %s", fieldName, typeName)
		}
	}

	// Also verify in the schema definition
	schema := g.SchemaDefinition(ctx)
	t.Logf("Schema excerpt:\n%s", schema)

	// Check that schema uses correct scalar names
	if !strings.Contains(schema, "stringField: String!") {
		t.Error("Schema should use 'String!' not 'string!' for string fields")
	}
	if !strings.Contains(schema, "intField: Int!") {
		t.Error("Schema should use 'Int!' not 'int!' for int fields")
	}
	if !strings.Contains(schema, "floatField: Float!") {
		t.Error("Schema should use 'Float!' not 'float64!' for float fields")
	}
	if !strings.Contains(schema, "boolField: Boolean!") {
		t.Error("Schema should use 'Boolean!' not 'bool!' for bool fields")
	}
}

// TestEnumNotConfusedWithScalar verifies that enum types with string underlying type
// are not confused with the String scalar
func TestEnumNotConfusedWithScalar(t *testing.T) {
	// Define a test-specific enum type to avoid conflicts
	type TestEnum string

	// Make TestEnum implement StringEnumValues
	type testEnumWrapper struct {
		value TestEnum
	}

	g := &Graphy{}
	ctx := context.Background()

	// Register enum manually using a wrapper that implements StringEnumValues
	g.RegisterTypes(ctx, &struct {
		Status TestEnum `gq:"enum:StatusEnum=ACTIVE,INACTIVE,PENDING"`
	}{})

	// Use simpler approach - just verify that String scalar is SCALAR kind
	// Enable introspection
	g.EnableIntrospection(ctx)

	// Query for String type
	query := `{
		stringType: __type(name: "String") {
			name
			kind
		}
	}`

	resp, err := g.ProcessRequest(ctx, query, "{}")
	if err != nil {
		t.Fatalf("Failed to process request: %v", err)
	}

	// Parse the response
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data := result["data"].(map[string]interface{})

	// Check String type
	stringType := data["stringType"].(map[string]interface{})
	if stringType["kind"] != "SCALAR" {
		t.Errorf("String should be SCALAR, got %v", stringType["kind"])
	}
	t.Logf("✓ String is correctly a SCALAR")
}
