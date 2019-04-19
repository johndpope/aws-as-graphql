const lambdaPlayground = require('graphql-playground-middleware-lambda')

module.exports.handler = lambdaPlayground({
	endpoint: '/dev/graphql',
});
