package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/testutil"
)

// Response is of type APIGatewayProxyResponse since we're leveraging the
// AWS Lambda Proxy Request functionality (default behavior)
//
// https://serverless.com/framework/docs/providers/aws/events/apigateway/#lambda-proxy-integration
type Response events.APIGatewayProxyResponse

// Handler is our lambda handler invoked by the `lambda.Start` function call
func Handler(ctx context.Context) (Response, error) {
	var buf bytes.Buffer

	body, err := json.Marshal(map[string]interface{}{
		"message": "Go Serverless v1.0! Your function executed successfully!",
	})
	if err != nil {
		return Response{StatusCode: 404}, err
	}

	json.HTMLEscape(&buf, body)

	// sess, err := session.NewSession(&aws.Config{Region: aws.String("us-west-2")})

	resp := Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            buf.String(),
		Headers: map[string]string{
			"Content-Type":           "application/json",
			"X-MyCompany-Func-Reply": "hello-handler",
		},
	}

	return resp, nil
}

func normalizeGraphQLName(str string) string {
	return strings.Replace(str, ".", "_", -1)
}

func convertPrimitiveToGraphQLType(t reflect.Type) *graphql.Scalar {
	switch t.Kind() {
	case reflect.String:
		return graphql.String
	case reflect.Bool:
		return graphql.Boolean
	}

	return graphql.String
}

func createCustomType(t reflect.Type) *graphql.Object {
	fields := graphql.Fields{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// fmt.Printf("Name: %v, Type: %v, Kind: %v ", field.Name, field.Type, field.Type.Kind())

		if field.Type.Kind() == reflect.Ptr {
			// fmt.Printf("Elem: %v, %v\n", field.Type.Elem(), field.Type.Elem().Kind())

			fields[field.Name] = &graphql.Field{}

			if field.Type.Elem().Kind() == reflect.Struct {
				fields[field.Name].Type = createCustomType(field.Type.Elem())
			} else if field.Tag.Get("required") == "true" {
				fields[field.Name].Type = graphql.NewNonNull(convertPrimitiveToGraphQLType(field.Type.Elem()))
			} else {
				fields[field.Name].Type = convertPrimitiveToGraphQLType(field.Type.Elem())
			}
		}
	}

	if len(fields) == 0 {
		fields["Empty"] = &graphql.Field{
			Description: "NOT YET IMPLEMENTED (probably array or number)",
			Type:        graphql.Boolean,
		}
	}

	return graphql.NewObject(graphql.ObjectConfig{
		Name:   normalizeGraphQLName(t.String()),
		Fields: fields,
	})
}

func createCustomInputType(t reflect.Type) *graphql.InputObject {
	fields := graphql.InputObjectConfigFieldMap{}

	// fmt.Printf("IN - Kind: %v, Elem: %v\n", t.String(), t.Kind())

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// fmt.Printf("Name: %v, Type: %v, Kind: %v ", field.Name, field.Type, field.Type.Kind())

		if field.Type.Kind() == reflect.Ptr {
			// fmt.Printf("Elem: %v, %v\n", field.Type.Elem(), field.Type.Elem().Kind())

			fields[field.Name] = &graphql.InputObjectFieldConfig{}

			if field.Type.Elem().Kind() == reflect.Struct {
				fields[field.Name].Type = createCustomInputType(field.Type.Elem())
			} else if field.Tag.Get("required") == "true" {
				fields[field.Name].Type = graphql.NewNonNull(convertPrimitiveToGraphQLType(field.Type.Elem()))
			} else {
				fields[field.Name].Type = convertPrimitiveToGraphQLType(field.Type.Elem())
			}
		}
	}

	if len(fields) == 0 {
		fields["Empty"] = &graphql.InputObjectFieldConfig{
			Description: "NOT YET IMPLEMENTED (probably array or number)",
			Type:        graphql.Boolean,
		}
	}

	return graphql.NewInputObject(
		graphql.InputObjectConfig{
			Name:   normalizeGraphQLName(t.String()),
			Fields: fields,
		},
	)
}

func parseInputArg(t reflect.Type) graphql.FieldConfigArgument {
	fields := graphql.FieldConfigArgument{}

	if t.Kind() == reflect.Struct {
		fields[normalizeGraphQLName(t.String())] = &graphql.ArgumentConfig{
			Type: graphql.NewNonNull(createCustomInputType(t)),
		}
	} else {
		fields[normalizeGraphQLName(t.String())] = &graphql.ArgumentConfig{
			Type: graphql.NewNonNull(convertPrimitiveToGraphQLType(t)),
		}
	}

	return fields
}

func parseOutputArg(t reflect.Type) *graphql.Object {
	fields := graphql.Fields{}

	if t.Kind() == reflect.Struct {
		fields[normalizeGraphQLName(t.String())] = &graphql.Field{
			Type: graphql.NewNonNull(createCustomType(t)),
		}
	} else {
		fields[normalizeGraphQLName(t.String())] = &graphql.Field{
			Type: graphql.NewNonNull(convertPrimitiveToGraphQLType(t)),
		}
	}

	out := graphql.NewObject(graphql.ObjectConfig{
		Name:   normalizeGraphQLName(t.String()) + "_return",
		Fields: fields,
	})

	return out
}

func parseFunction(t reflect.Type) *graphql.Field {
	field := &graphql.Field{}
	field.Description = t.Name()

	// Parse Input Arguments
	numIn := t.NumIn()
	for i := 0; i < numIn; i++ {
		inV := t.In(i)
		field.Args = parseInputArg(inV.Elem())
	}

	// Parse Output Values
	numOut := t.NumOut()
	for o := 0; o < numOut; o++ {
		// returnV := t.Out(o)
		// fmt.Printf("OUT - Kind: %v Name: %v String: %v\n", returnV.Kind(), returnV.Name(), returnV.String())

	}

	field.Type = parseOutputArg(t.Out(0).Elem())

	return field

}

func main() {
	sess, err := session.NewSession()
	if err != nil {
		// Handle Session creation error
	}
	svc := s3.New(sess)
	queryFields := graphql.Fields{}
	mutationFields := graphql.Fields{}

	s3Type := reflect.TypeOf(svc)
	for i := 0; i < s3Type.NumMethod(); i++ {
		method := s3Type.Method(i)
		fmt.Println(method.Name)

		// if strings.HasPrefix(method.Name, "Create") || strings.HasPrefix(method.Name, "Delete") {
		// 	mutationFields[method.Name] = parseFunction(s3Type.Method(i).Type)
		// } else
		if strings.HasPrefix(method.Name, "Put") && !strings.HasSuffix(method.Name, "Request") && !strings.HasSuffix(method.Name, "WithContext") {
			queryFields[method.Name] = parseFunction(s3Type.Method(i).Type)
		}
	}

	// x := reflect.TypeOf(svc.CreateBucket)

	// fields["createBucket"] = parseFunction(x)

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

	// lambda.Start(Handler)
}
