![alt text](https://i.kym-cdn.com/photos/images/newsfeed/000/114/139/tumblr_lgedv2Vtt21qf4x93o1_40020110725-22047-38imqt.jpg)


### AWS as GraphQL

##### What???

Expose you AWS Account as GraphQL Endpoint. Use it instead of CLI or SDK.

##### Why?

AWS UI/UX is imperfect in some places. With this endpoint, creating custom dashboard should be much faster/easier.

##### How?

Project is using reflection to get all methods from AWS SDK and convert them into GraphQL schema.
