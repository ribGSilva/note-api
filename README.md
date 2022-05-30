# Note-API

## Objective

This is an app to test abstractions and tests.

### Tests

To run the tests just run

    make tests

### Env setup

Use the make file commands to set up your env:

    make env-build
    make env-setup

To shutdown your env, just call:

    make env-down
    make env-clear

### Migrations

The make command 'env-setup' already handle the schema creation, but you can access the binary to see all commands available:

    go run ./app/cmd/main.go help

## Features

- Idempotency: IDEMPOTENCY_ENABLED
- NewRelic integration: NEW_RELIC_ENABLED, NEW_RELIC_LICENCE, NEW_RELIC_APP_NAME
- Swagger: SWAGGER_HOST, SWAGGER_PROTOCOL 

### Arch

    app -> accessable interfaces as apis, commands, message listeners, so on...
    business -> business logic
    persistence -> all about store and retrieve data, no matter if is an api or a database
    plathform -> usually stays is a priv lib
    sys -> holds configurations and app resources
    zarf -> has configurations files, usefull binaries, and so on... 

### k8s

To run inside k8s

I used kind -> https://kind.sigs.k8s.io/

Ps: Install if and start a cluster first

Generate a docker image:

    make docker-build

Upload the image to the kind cluster:

    kind load docker-image note-api:1.0

Put the env vars in the app:

    zarf/k8s/note-api/deployment.yaml

Apply the services into kind:

    kubectl apply -f ./zarf/k8s/note-api

Start a bridge to the app:

    kubectl port-forward svc/note-api-service 8080:80

And access the swagger on your browser:

    http://localhost:8080/swagger/index.html

And it is done 🥳