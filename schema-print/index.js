const { printSchema, buildClientSchema } = require('graphql')

const { data } = require('../schema.json');
const schema = buildClientSchema(data);
console.log(printSchema(schema));