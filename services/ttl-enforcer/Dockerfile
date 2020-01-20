ARG TERRAFORM_PLUGIN_DIR=/root/.terraform.d/plugins/linux_amd64

FROM golang:1.13 as builder

COPY ttl-enforcer/ /lt/ttl-enforcer/

WORKDIR /lt/ttl-enforcer
RUN GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build --installsuffix cgo --ldflags="-s" -o ttl-enforcer

FROM python:3.8-alpine
ARG TERRAFORM_PLUGIN_DIR
ENV CHECKPOINT_DISABLE true
ENV TFSTATE_DIR_PATH /tfstate
ENV SCRIPT_DIR_PATH /terraform
RUN apk add --no-cache curl bash
RUN pip install boto
RUN pip install boto3
RUN pip install awscli
RUN mkdir /artifacts
RUN curl https://releases.hashicorp.com/terraform/0.12.20/terraform_0.12.20_linux_amd64.zip -o /artifacts/terraform.zip
RUN unzip /artifacts/terraform.zip -d /usr/local/bin
RUN mkdir -p $TERRAFORM_PLUGIN_DIR
RUN curl https://releases.hashicorp.com/terraform-provider-aws/2.48.0/terraform-provider-aws_2.48.0_linux_amd64.zip -o /artifacts/aws.zip
RUN unzip /artifacts/aws.zip -d $TERRAFORM_PLUGIN_DIR 
RUN curl https://releases.hashicorp.com/terraform-provider-tls/2.1.1/terraform-provider-tls_2.1.1_linux_amd64.zip -o /artifacts/tls.zip
RUN unzip /artifacts/tls.zip -d $TERRAFORM_PLUGIN_DIR 
RUN rm -rf /artifacts
COPY terraform/ $SCRIPT_DIR_PATH/
RUN mkdir $TFSTATE_DIR_PATH
ENV SHELL=/bin/bash
COPY --from=builder /lt/ttl-enforcer/ttl-enforcer /lt/ttl-enforcer/ttl-enforcer
COPY ttl-enforcer/settings.yaml /lt/ttl-enforcer/settings.yaml
RUN apk update && apk add --no-cache git ca-certificates
COPY ttl-enforcer/docker-entrypoint.sh /usr/local/bin/
RUN chmod 777 /usr/local/bin/docker-entrypoint.sh && ln -s /usr/local/bin/docker-entrypoint.sh /
ENTRYPOINT ["docker-entrypoint.sh"]
WORKDIR /lt/ttl-enforcer
CMD ["/lt/ttl-enforcer/ttl-enforcer"]
