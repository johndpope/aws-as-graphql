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
	"strconv"
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
	fields := &graphql.ArgumentConfig{}

	fmt.Printf("IN - Kind: %v, Elem: %v\n", t.String(), t.Kind())

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		fmt.Printf("Name: %v, Type: %v, Kind: %v ", field.Name, field.Type, field.Type.Kind())

		if field.Type.Kind() == reflect.Ptr {
			fmt.Printf("Elem: %v, %v\n", field.Type.Elem(), field.Type.Elem().Kind())

			if field.Type.Elem().Kind() == reflect.Struct {
				fields.Type = createCustomType(field.Type.Elem())
			} else if field.Tag.Get("required") == "true" {
				fields.Type = graphql.NewNonNull(convertPrimitiveToGraphQLType(field.Type.Elem()))
			} else {
				fields.Type = graphql.NewNonNull(convertPrimitiveToGraphQLType(field.Type.Elem()))
			}
		}
	}

	return graphql.NewObject(graphql.ObjectConfig{
		Name:   t.String(),
		Fields: fields,
	})
}

func parseInputArg(t reflect.Type) graphql.FieldConfigArgument {
	fields := graphql.FieldConfigArgument{}

	if t.Kind() == reflect.Struct {
		fields[t.String()] = &graphql.ArgumentConfig{
			Type: graphql.NewNonNull(createCustomType(t)),
		}
	} else {
		fields[t.String()] = &graphql.ArgumentConfig{
			Type: convertPrimitiveToGraphQLType(t),
		}
	}

	return fields
}

func parseOutputArg(t reflect.Type) *graphql.Object {
	ss := strings.Split(t.String(), ".")
	name := ss[len(ss)-1]

	out := graphql.NewObject(graphql.ObjectConfig{
		Name:   name,
		Fields: graphql.Fields{},
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
		returnV := t.Out(o)
		fmt.Printf("\nParameter OUT: "+strconv.Itoa(o)+"\nKind: %v Name: %v String: %v", returnV.Kind(), returnV.Name(), returnV.String())

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

	// s3Type := reflect.TypeOf(svc)
	// for i := 0; i < s3Type.NumMethod(); i++ {
	// 	method := s3Type.Method(i)
	// 	fmt.Println(method.Name)
	// }

	x := reflect.TypeOf(svc.CreateBucket)

	fields := graphql.Fields{
		"createBucket": parseFunction(x),
	}

	rootQuery := graphql.ObjectConfig{Name: "RootQuery", Fields: fields}
	rootMutation := graphql.ObjectConfig{Name: "RootMutation", Fields: fields}

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
