package main

import (
	"errors"

	"github.com/graphql-go/graphql"
)

var processedTypes = make(map[string]*graphql.Object)
var processedInputTypes = make(map[string]*graphql.InputObject)

func lookupInputType(s string) (*graphql.InputObject, error) {
	if val, ok := processedInputTypes[s]; ok {
		return val, nil
	}

	return nil, errors.New("NOT_FOUND")
}

func lookupType(s string) (*graphql.Object, error) {
	if val, ok := processedTypes[s]; ok {
		return val, nil
	}

	return nil, errors.New("NOT_FOUND")
}

func markTypeAsProcessed(s string, item *graphql.Object) {
	processedTypes[s] = item
}

func markInputTypeAsProcessed(s string, item *graphql.InputObject) {
	processedInputTypes[s] = item
}
