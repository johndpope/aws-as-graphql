const lambdaPlayground = require('graphql-playground-middleware-lambda');

const playgroundHandler = lambdaPlayground({
	endpoint: '/dev/graphql',
});

module.exports = { playgroundHandler };
