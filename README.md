# AWS as GraphQL

### What???

Expose you AWS Account as GraphQL Endpoint. Use it instead of CLI or SDK.

### Why?

AWS UI/UX is imperfect in some places. With this endpoint, creating custom dashboard should be much faster/easier.

### How?

Project is using reflection to get all methods from AWS SDK and convert them into GraphQL schema.

## Prerequisites
- Node.js 8
- Go 1.x
- Serverless Framework

## Deploying
```sh
sls deploy
```
