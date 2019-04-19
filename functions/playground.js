const lambdaPlayground = require('graphql-playground-middleware-lambda').default

module.exports.handler = lambdaPlayground({
	endpoint: '/dev/graphql',
});
