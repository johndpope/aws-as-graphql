package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strings"

	typesCache "github.com/RafalWilinski/aws-as-graphql/typescache"
	"github.com/RafalWilinski/aws-as-graphql/utils"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/testutil"
)

func createCustomType(t reflect.Type) *graphql.Object {
	t = utils.NormalizePointerType(t)
	name := utils.NormalizeGraphQLName(t.String())

	obj, err := typesCache.LookupType(name)
	if err == nil {
		return obj
	}

	fields := graphql.Fields{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Name == "_" || field.Name == "__" {
			continue
		}

		fields[field.Name] = &graphql.Field{}

		fieldType := utils.NormalizePointerType(field.Type)
		fieldKind := fieldType.Kind()

		var graphqlType graphql.Output

		if fieldKind == reflect.Struct {
			graphqlType = createCustomType(fieldType)
		} else if fieldKind == reflect.Slice {
			sliceElement := utils.NormalizePointerType(fieldType.Elem())

			if sliceElement.Kind() == reflect.Struct {
				graphqlType = graphql.NewList(createCustomType(sliceElement))
			} else {
				graphqlType = graphql.NewList(utils.ConvertPrimitiveToGraphQLType(sliceElement))
			}
		} else if fieldKind == reflect.Interface {
			continue
		} else {
			graphqlType = utils.ConvertPrimitiveToGraphQLType(fieldType)
		}

		if field.Tag.Get("required") == "true" {
			fields[field.Name].Type = graphql.NewNonNull(graphqlType)
		} else {
			fields[field.Name].Type = graphqlType
		}
	}

	if len(fields) == 0 {
		fields["Empty"] = &graphql.Field{
			Description: "NOT YET IMPLEMENTED (probably array)",
			Type:        graphql.Boolean,
		}
	}

	object := graphql.NewObject(graphql.ObjectConfig{
		Name:   name,
		Fields: fields,
	})

	typesCache.MarkTypeAsProcessed(name, object)

	return object
}

func createCustomInputType(t reflect.Type) *graphql.InputObject {
	t = utils.NormalizePointerType(t)
	name := utils.NormalizeInputName(utils.NormalizeGraphQLName(t.String()))
	obj, err := typesCache.LookupInputType(name)
	if err == nil {
		return obj
	}

	fields := graphql.InputObjectConfigFieldMap{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Name == "_" {
			continue
		}

		fields[field.Name] = &graphql.InputObjectFieldConfig{}

		fieldType := utils.NormalizePointerType(field.Type)
		fieldKind := fieldType.Kind()

		var graphqlType graphql.Output

		if fieldKind == reflect.Struct {
			graphqlType = createCustomInputType(fieldType)
		} else if fieldKind == reflect.Slice {
			sliceElement := utils.NormalizePointerType(fieldType.Elem())

			if sliceElement.Kind() == reflect.Struct {
				graphqlType = graphql.NewList(createCustomInputType(sliceElement))
			} else {
				graphqlType = graphql.NewList(utils.ConvertPrimitiveToGraphQLType(sliceElement))
			}
		} else if fieldKind == reflect.Interface {
			graphqlType = graphql.String
		} else {
			graphqlType = utils.ConvertPrimitiveToGraphQLType(fieldType)
		}

		if field.Tag.Get("required") == "true" {
			fields[field.Name].Type = graphql.NewNonNull(graphqlType)
		} else {
			fields[field.Name].Type = graphqlType
		}
	}

	if len(fields) == 0 {
		fields["Empty"] = &graphql.InputObjectFieldConfig{
			Description: "NOT YET IMPLEMENTED (probably array)",
			Type:        graphql.Boolean,
		}
	}

	inputObject := graphql.NewInputObject(
		graphql.InputObjectConfig{
			Name:   name,
			Fields: fields,
		},
	)

	typesCache.MarkInputTypeAsProcessed(name, inputObject)

	return inputObject
}

func parseInputArg(t reflect.Type) graphql.FieldConfigArgument {
	fields := graphql.FieldConfigArgument{}

	if t.Kind() == reflect.Struct {
		name := utils.NormalizeInputName(utils.NormalizeGraphQLName(t.String()))
		obj, err := typesCache.LookupInputType(name)
		if err == nil {
			fields["data"] = &graphql.ArgumentConfig{
				Type: graphql.NewNonNull(obj),
			}
		}

		inputType := createCustomInputType(t)
		typesCache.MarkInputTypeAsProcessed(name, inputType)

		fields["data"] = &graphql.ArgumentConfig{
			Type: graphql.NewNonNull(inputType),
		}
	} else {
		fields[utils.NormalizeGraphQLName(t.String())] = &graphql.ArgumentConfig{
			Type: graphql.NewNonNull(utils.ConvertPrimitiveToGraphQLType(t)),
		}
	}

	return fields
}

func parseOutputArg(t reflect.Type) *graphql.Object {
	fields := graphql.Fields{}

	if t.Kind() == reflect.Struct {
		fields[utils.NormalizeGraphQLName(t.String())] = &graphql.Field{
			Type: graphql.NewNonNull(createCustomType(t)),
		}
	} else {
		fields[utils.NormalizeGraphQLName(t.String())] = &graphql.Field{
			Type: graphql.NewNonNull(utils.ConvertPrimitiveToGraphQLType(t)),
		}
	}

	out := graphql.NewObject(graphql.ObjectConfig{
		Name:   utils.NormalizeGraphQLName(t.String()) + "Response",
		Fields: fields,
	})

	return out
}

func parseFunction(t reflect.Type) *graphql.Field {
	field := &graphql.Field{}
	field.Description = t.Name()

	numIn := t.NumIn()
	for i := 0; i < numIn; i++ {
		inV := t.In(i)
		if inV.Kind() == reflect.Func || inV.Kind() == reflect.Interface {
			continue
		}

		field.Args = parseInputArg(inV.Elem())
	}
	outParam := utils.NormalizePointerType(t.Out(0))

	fmt.Printf("%v, %v\n", outParam.String(), outParam.Kind().String())

	if outParam.Kind() == reflect.Struct {
		field.Type = parseOutputArg(outParam)
	} else {
		fmt.Printf("Unrecognized out param! %v, %v\n", outParam.String(), outParam.Kind().String())
		field.Type = graphql.Boolean
	}

	return field

}

func buildSchema() {
	sess, _ := session.NewSession()
	svc := s3.New(sess)
	queryFields := graphql.Fields{}
	mutationFields := graphql.Fields{}

	s3Type := reflect.TypeOf(svc)
	for i := 0; i < s3Type.NumMethod(); i++ {
		method := s3Type.Method(i)

		// Filter redundant functions and helper functions
		if strings.HasSuffix(method.Name, "Request") || strings.HasSuffix(method.Name, "WithContext") {
			continue
		}

		if strings.HasPrefix(method.Name, "Create") || strings.HasPrefix(method.Name, "Delete") || strings.HasPrefix(method.Name, "Put") {
			mutationFields[method.Name] = parseFunction(s3Type.Method(i).Type)
		} else if strings.HasPrefix(method.Name, "Get") || strings.HasPrefix(method.Name, "List") || strings.HasPrefix(method.Name, "Describe") {
			queryFields[method.Name] = parseFunction(s3Type.Method(i).Type)
		}
	}

	rootQuery := graphql.ObjectConfig{Name: "RootQuery", Fields: queryFields}
	rootMutation := graphql.ObjectConfig{Name: "RootMutation", Fields: mutationFields}

	schemaConfig := graphql.SchemaConfig{
		Query:    graphql.NewObject(rootQuery),
		Mutation: graphql.NewObject(rootMutation),
	}

	schema, err := graphql.NewSchema(schemaConfig)

	if err != nil {
		log.Fatalf("failed to create new schema, error: %v", err)
	}

	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: testutil.IntrospectionQuery,
	})

	if result.HasErrors() {
		log.Fatalf("ERROR introspecting schema: %v", result.Errors)
	} else {
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			log.Fatalf("ERROR: %v", err)
		}
		err = ioutil.WriteFile("./schema.json", b, os.ModePerm)
		if err != nil {
			log.Fatalf("ERROR: %v", err)
		}
	}
}
