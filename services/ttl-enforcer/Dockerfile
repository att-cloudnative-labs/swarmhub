FROM golang:1.12 as builder

COPY main.go /lt/ttl-enforcer/main.go
COPY go.mod /lt/ttl-enforcer/go.mod
COPY go.sum /lt/ttl-enforcer/go.sum

WORKDIR /lt/ttl-enforcer
RUN GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build --installsuffix cgo --ldflags="-s" -o ttl-enforcer

FROM alpine:3.9
COPY --from=builder /lt/ttl-enforcer/ttl-enforcer /lt/ttl-enforcer/ttl-enforcer
COPY settings.yaml /lt/ttl-enforcer/settings.yaml

RUN apk update && apk add --no-cache git ca-certificates
WORKDIR /lt/ttl-enforcer

ENTRYPOINT ["/lt/ttl-enforcer/ttl-enforcer"]
