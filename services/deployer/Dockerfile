ARG TERRAFORM_PLUGIN_DIR=/root/.terraform.d/plugins/linux_amd64
ARG TF_CLI_CONFIG_FILE=/terraform/.terraformrc

FROM golang:1.13 AS builder
COPY src/ /src/
WORKDIR /src/deployer
RUN GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build --installsuffix cgo --ldflags="-s" -o /main

FROM python:3.8-alpine
ARG TERRAFORM_PLUGIN_DIR
ARG TF_CLI_CONFIG_FILE
ENV TF_CLI_CONFIG_FILE $TF_CLI_CONFIG_FILE
RUN apk add --no-cache bash curl unzip
RUN pip install boto
RUN pip install boto3
RUN pip install awscli
RUN ln -s /usr/local/bin/python /usr/bin/python
RUN mkdir /artifacts
RUN curl https://releases.hashicorp.com/terraform/0.12.20/terraform_0.12.20_linux_amd64.zip -o /artifacts/terraform.zip
RUN unzip /artifacts/terraform.zip -d /usr/local/bin
RUN curl -iL --max-redirs 1 https://github.com/rancher/rke/releases/download/v1.0.4/rke_linux-amd64 -o /usr/local/bin/rke
RUN chmod 755 -R /usr/local/bin/rke
RUN mkdir -p $TERRAFORM_PLUGIN_DIR
RUN curl https://releases.hashicorp.com/terraform-provider-aws/2.48.0/terraform-provider-aws_2.48.0_linux_amd64.zip -o /artifacts/aws.zip
RUN unzip /artifacts/aws.zip -d $TERRAFORM_PLUGIN_DIR 
RUN curl https://releases.hashicorp.com/terraform-provider-tls/2.1.1/terraform-provider-tls_2.1.1_linux_amd64.zip -o /artifacts/tls.zip
RUN unzip /artifacts/tls.zip -d $TERRAFORM_PLUGIN_DIR 
RUN curl https://releases.hashicorp.com/terraform-provider-null/2.1.2/terraform-provider-null_2.1.2_linux_amd64.zip -o /artifacts/null.zip
RUN unzip /artifacts/null.zip -d $TERRAFORM_PLUGIN_DIR
RUN curl https://releases.hashicorp.com/terraform-provider-local/1.4.0/terraform-provider-local_1.4.0_linux_amd64.zip -o /artifacts/local.zip
RUN unzip /artifacts/local.zip -d $TERRAFORM_PLUGIN_DIR
RUN curl -iL --max-redirs 1 https://github.com/rancher/terraform-provider-rke/releases/download/0.14.1/terraform-provider-rke_0.14.1_linux-amd64.zip -o /artifacts/rke
RUN unzip /artifacts/rke -d $TERRAFORM_PLUGIN_DIR ; exit 0
RUN curl -iL --max-redirs 1 https://github.com/terraform-providers/terraform-provider-kubernetes/releases/download/v1.10.0/terraform-provider-kubernetes_1.10.0_linux_amd64.zip -o /artifacts/k8
RUN unzip /artifacts/k8 -d $TERRAFORM_PLUGIN_DIR ; exit 0
ENV SHELL=/bin/bash
RUN rm -rf /artifacts
COPY --from=builder /main /main
COPY terraform/ /terraform/


CMD ["/main"]
