package quickgraph

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGraphy_introspection(t *testing.T) {
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

     fields(includeDeprecated: false) {
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
     enumValues(includeDeprecated: false) {
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

	result, err := g.ProcessRequest(ctx, query, "")
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
              "name": "Character",
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
              "name": "Character",
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
              "name": "Character",
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
              "name": "Character",
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
              "name": "Character",
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
              "name": "ConnectionEdge",
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
              "name": "Droid",
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
              "name": "Droid",
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
              "name": "Droid",
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
              "name": "Droid",
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
              "name": "Droid",
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
              "name": "Droid",
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
              "name": "FriendsConnection",
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
              "name": "FriendsConnection",
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
              "name": "Human",
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
              "name": "Human",
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
              "name": "Human",
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
              "name": "Human",
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
              "name": "Human",
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
              "name": "Human",
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
              "name": "Human",
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
              "name": "Human",
              "ofType": null
            },
            {
              "kind": "OBJECT",
              "name": "Droid",
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
              "name": "Starship",
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
              "name": "Starship",
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
