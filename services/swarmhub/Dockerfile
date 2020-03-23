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
RUN apk update && apk add ca-certificates curl unzip && rm -rf /var/cache/apk/*
RUN update-ca-certificates
RUN mkdir -p /app
RUN mkdir -p /tfstate
ENV TFSTATE_DIR_PATH /tfstate
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

RUN curl https://releases.hashicorp.com/terraform/0.12.20/terraform_0.12.20_linux_amd64.zip -o terraform.zip
RUN unzip terraform.zip -d /usr/local/bin && rm terraform.zip

ENTRYPOINT ["/app/main"]
