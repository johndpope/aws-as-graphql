package typesCache

import (
	"errors"

	"github.com/graphql-go/graphql"
)

var processedTypes = make(map[string]*graphql.Object)
var processedInputTypes = make(map[string]*graphql.InputObject)

// LookupInputType returns existing GraphQLInputObject if such was processed already
func LookupInputType(s string) (*graphql.InputObject, error) {
	if val, ok := processedInputTypes[s]; ok {
		return val, nil
	}

	return nil, errors.New("NOT_FOUND")
}

// LookupType returns existing GraphQLObject if such was processed already
func LookupType(s string) (*graphql.Object, error) {
	if val, ok := processedTypes[s]; ok {
		return val, nil
	}

	return nil, errors.New("NOT_FOUND")
}

// MarkTypeAsProcessed adds GraphQLObject to store of processes types
func MarkTypeAsProcessed(s string, item *graphql.Object) {
	processedTypes[s] = item
}

// MarkTypeAsProcessed adds GraphQLInputObject to store of processes Input types
func MarkInputTypeAsProcessed(s string, item *graphql.InputObject) {
	processedInputTypes[s] = item
}
