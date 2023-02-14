module github.com/michaelhenkel/contrail-cni

go 1.13

replace github.com/docker/distribution v2.7.1+incompatible => github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible

require (
	github.com/containernetworking/cni v0.7.1
	github.com/containernetworking/plugins v0.8.2
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.3.2
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/vishvananda/netlink v1.0.0
	golang.org/x/time v0.0.0-20190921001708-c4c64cad1fd0 // indirect
	google.golang.org/grpc v1.24.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	k8s.io/api v0.0.0-20191010143144-fbf594f18f80 // indirect
	k8s.io/apimachinery v0.0.0-20191014065749-fb3eea214746
	k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
	k8s.io/utils v0.0.0-20191010214722-8d271d903fe4 // indirect
)
