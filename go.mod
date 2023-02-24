module github.com/michaelhenkel/contrail-cni

go 1.13

replace github.com/docker/distribution v2.7.1+incompatible => github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible

require (
	github.com/containernetworking/cni v0.7.1
	github.com/containernetworking/plugins v0.8.2
	github.com/golang/protobuf v1.4.2
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/vishvananda/netlink v1.0.0
	google.golang.org/grpc v1.27.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	k8s.io/apimachinery v0.20.0-alpha.2
	k8s.io/client-go v0.20.0-alpha.2
)
