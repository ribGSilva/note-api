FROM golang:1.18.2-alpine AS build_base

RUN apk add --no-cache ca-certificates
RUN wget https://github.com/upx/upx/releases/download/v3.96/upx-3.96-amd64_linux.tar.xz && \
    tar -xvf *.xz && cd upx* && cp upx /usr/bin

WORKDIR /go/src/service

COPY ../../go.mod ./go.mod
COPY ../../go.sum ./go.sum

#COPY vendor ./vendor
RUN go mod download

COPY ../.. .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o application ./app/messaging/main.go && \
     upx --lzma /go/src/service/application

FROM scratch

USER 1000
COPY --chown=1000 --from=build_base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --chown=1000 --from=build_base /go/src/service/application /application
EXPOSE 8080

CMD ["./application"]