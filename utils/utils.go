package utils

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/graphql-go/graphql"
)

func NormalizeGraphQLName(str string) string {
	return strings.Replace(str, ".", "_", -1)
}

func NormalizeInputName(str string) string {
	if strings.HasSuffix(str, "Input") {
		return str
	}

	return str + "Input"
}

func ConvertPrimitiveToGraphQLType(t reflect.Type) *graphql.Scalar {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return graphql.String
	case reflect.Bool:
		return graphql.Boolean
	case reflect.Int:
		return graphql.Int
	case reflect.Int16:
		return graphql.Int
	case reflect.Int32:
		return graphql.Int
	case reflect.Int64:
		return graphql.Int
	case reflect.Uint8:
		return graphql.Int
	case reflect.Uint16:
		return graphql.Int
	case reflect.Uint32:
		return graphql.Int
	case reflect.Uint64:
		return graphql.Int
	}

	fmt.Printf("Unhandled primitive type: %v\n", t.Kind().String())

	return graphql.String
}

func NormalizePointerType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		return t.Elem()
	}

	return t
}
