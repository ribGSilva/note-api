FROM golang:1.18.2-alpine AS build_stage

RUN apk add --no-cache ca-certificates
RUN wget https://github.com/upx/upx/releases/download/v3.96/upx-3.96-amd64_linux.tar.xz && \
    tar -xvf *.xz && cd upx* && cp upx /usr/bin

WORKDIR /go/src/service

COPY ../../go.mod ./go.mod
COPY ../../go.sum ./go.sum

#COPY vendor ./vendor
RUN go mod download

COPY ../.. .

RUN CGO_ENABLED=0 go test -coverprofile ./coverage-report.out -coverpkg=./... -json ./... > ./test-report.json

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o application ./app/api/main.go && \
     upx --lzma /go/src/service/application

FROM sonarsource/sonar-scanner-cli AS test_stage

ARG SKIP_TESTS=false
ARG SONAR_HOST_URL
ARG SONAR_LOGIN

COPY --from=build_stage /go/src/service/ ./

RUN if [ $SKIP_TESTS != true ]; \
    then sonar-scanner \
        -Dsonar.host.url=$SONAR_HOST_URL \
        -Dsonar.login=$SONAR_LOGIN \
        -Dproject.settings=./sonar-project.properties; \
    fi;

FROM scratch

USER 1000
COPY --chown=1000 --from=build_stage /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --chown=1000 --from=test_stage /usr/src/application /application
EXPOSE 8080

CMD ["./application"]