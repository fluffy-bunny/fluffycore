# OAuth2 client_credentials server

The example app contains an OAuth2 server that has an external [client.json](../docker/config/example/clients.json) file that can be mounted via a docker-compose volume.  This useful for testing authorization apis using the JWTs it creates.  

You can spin up the OAuth2 server in you own apps using this [self contained service](../mocks/oauth2/echo/server.go).  [example](.../example/internal/runtime/startup.go#L115)  


```bash
cd docker
docker-compose up

```

[Discovery Document](http://localhost:50053/.well-known/openid-configuration)  
[JWKS](http://localhost:50053/.well-known/jwks)  
