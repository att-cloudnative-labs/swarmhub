FROM golang:1.13 AS builder
COPY . /app
WORKDIR /app
RUN GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build --installsuffix cgo --ldflags="-s" -o /main

FROM alpine:3.9
COPY --from=builder /main /main
CMD ["/main"]
