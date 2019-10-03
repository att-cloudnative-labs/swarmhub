FROM golang:1.12 AS builder
COPY src/ /src/
WORKDIR /src/deployer
RUN GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build --installsuffix cgo --ldflags="-s" -o /main

WORKDIR /src/locust-go
RUN GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build --installsuffix cgo --ldflags="-s" -o /locust-go

FROM python:3.7.4-alpine3.10
COPY --from=builder /main /main
COPY hosts /etc/ansible/hosts
COPY ansible /ansible
COPY --from=builder /locust-go /ansible/roles/locust-deploy-test/files/locust-go
RUN apk add ansible
RUN apk add bash
RUN apk add rsync
RUN apk add openssh
RUN apk add openssl
RUN pip install boto
RUN pip install boto3
RUN ln -s /usr/local/bin/python /usr/bin/python

CMD ["/main"]
