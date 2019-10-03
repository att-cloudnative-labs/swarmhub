module github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub

go 1.12

require (
	github.com/aws/aws-sdk-go v1.19.31
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/julienschmidt/httprouter v1.2.0
	github.com/lib/pq v1.1.1
	github.com/nats-io/nats.go v1.8.1
	github.com/nats-io/stan.go v0.5.0
	github.com/prometheus/client_golang v0.9.3
	github.com/prometheus/common v0.4.0
	github.com/spf13/viper v1.4.0
	golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2
	gopkg.in/asn1-ber.v1 v1.0.0-20181015200546-f715ec2f112d // indirect
	gopkg.in/ldap.v2 v2.5.1
)
