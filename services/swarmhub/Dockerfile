FROM golang:1.12 as builder1
COPY src/swarmhub/go.mod /app/swarmhub/go.mod
COPY src/swarmhub/go.sum /app/swarmhub/go.sum
WORKDIR /app/swarmhub
RUN go mod download

FROM builder1 as builder2
COPY src/swarmhub/ /app/swarmhub
WORKDIR /app/swarmhub
RUN GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build --installsuffix cgo --ldflags="-s" -o /main

FROM node:10 as builder3
COPY frontend/package*.json ./
RUN npm install 

FROM builder3 as builder4
COPY frontend/ ./
RUN npm run build

FROM alpine:latest
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
RUN update-ca-certificates
RUN mkdir -p /app
WORKDIR /app
COPY --from=builder2 /main /app/main
COPY --from=builder2 /app/swarmhub/settings.yaml /app/settings.yaml
COPY --from=builder4 dist/index.html /var/www/swarmhub/html/index.html
COPY --from=builder4 dist/favicon.ico /var/www/swarmhub/html/favicon.ico
COPY --from=builder4 dist/static/ /var/www/swarmhub/static/
COPY --from=builder4 src/assets/attlogo.svg /var/www/swarmhub/static/images/attlogo.svg
COPY --from=builder4 static/html/login.html /var/www/swarmhub/html/login.html
COPY --from=builder4 static/html/logout.html /var/www/swarmhub/html/logout.html
COPY --from=builder4 static/html/base.html /var/www/swarmhub/html/base.html

ENTRYPOINT ["/app/main"]
