package quickgraph

import "reflect"

type GraphSchema struct {
	ResultTypes map[string]*TypeLookup
	Functions   map[string]*GraphFunction
	InputTypes  map[string]reflect.Type
	//SubscriptionFunctions map[string]*GraphFunction

}
