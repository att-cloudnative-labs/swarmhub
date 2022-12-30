module github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub

go 1.12

require (
	github.com/aws/aws-sdk-go v1.33.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/julienschmidt/httprouter v1.2.0
	github.com/lib/pq v1.10.4
	github.com/nats-io/nats-server/v2 v2.9.10 // indirect
	github.com/nats-io/nats-streaming-server v0.25.2 // indirect
	github.com/nats-io/nats.go v1.19.0
	github.com/nats-io/stan.go v0.10.3
	github.com/prometheus/client_golang v0.9.3
	github.com/prometheus/common v0.4.0
	github.com/spf13/viper v1.4.0
	golang.org/x/crypto v0.0.0-20221010152910-d6f0a8c073c2
	gopkg.in/asn1-ber.v1 v1.0.0-20181015200546-f715ec2f112d // indirect
	gopkg.in/ldap.v2 v2.5.1
)
