const { join } = require('path');
const { readFileSync } = require('fs')
const { printSchema, buildClientSchema } = require('graphql')

const schemaFile = readFileSync(join(__dirname, '..', 'schema.json'), 'utf8');
const schema = buildClientSchema(schemaFile);
console.log(printSchema(schema));