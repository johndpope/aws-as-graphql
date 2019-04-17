package main

import (
	"context"
	"encoding/json"

	schema "github.com/RafalWilinski/aws-as-graphql/schema"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/graphql-go/graphql"
)

type requestBody struct {
	Query          string                 `json:"query"`
	VariableValues map[string]interface{} `json:"variables"`
	OperationName  string                 `json:"operationName"`
}

func executeQuery(request requestBody, schema graphql.Schema) *graphql.Result {
	result := graphql.Do(graphql.Params{
		Schema:         schema,
		VariableValues: request.VariableValues,
		RequestString:  request.Query,
		OperationName:  request.OperationName,
	})

	return result
}

// Handler of Lambda function
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	requestBody := requestBody{}
	err := json.Unmarshal([]byte(request.Body), &requestBody)

	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 500}, err
	}

	schema, err := schema.BuildSchema()

	graphQLResult := executeQuery(requestBody, schema)
	responseJSON, err := json.Marshal(graphQLResult)

	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 400}, err
	}

	return events.APIGatewayProxyResponse{Body: string(responseJSON[:]), StatusCode: 200, Headers: map[string]string{
		"Content-Type": "application/json",
	}}, nil
}

func main() {
	lambda.Start(Handler)
}
