package main

import (
	"context"
	"fmt"
	"net"
	"strings"

	//"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"

	contrailCni "github.com/michaelhenkel/contrail-cni/contrail"
	pb "github.com/michaelhenkel/contrail-cni/contrailcni"

	//"github.com/michaelhenkel/contrail-cni/common"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types/current"
	log "github.com/michaelhenkel/contrail-cni/logging"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

//go:generate protoc -I ../contrailcni --go_out=plugins=grpc:../contrailcni ../contrailcni/contrailcni.proto

// Package main implements a server for Greeter service.

const (
	port = ":10000"
)

// server is used to implement contrailcni.ContrailCNIServer.
type server struct {
	pb.UnimplementedContrailCNIServer
}

func getCNI(ctx context.Context, in *pb.CNIArgs) (*contrailCni.ContrailCni, error) {
	skelArgs := getSkelArgs(in)
	cni, err := contrailCni.Init(skelArgs)
	if err != nil {
		return cni, err
	}

	argsList := strings.Split(skelArgs.Args, ";")
	var argMap = make(map[string]string)
	for _, arg := range argsList {
		argKV := strings.Split(arg, "=")
		argMap[argKV[0]] = argKV[1]
	}
	containerName := argMap["K8S_POD_NAME"]
	log.Info("containerName %s", containerName)
	uid, err := getPodIDFromKubeAPI(containerName, argMap["K8S_POD_NAMESPACE"])
	if err != nil {
		log.Error("couldn't get uid from kube %s\n", err)
	}
	containerUID := fmt.Sprintf("%s", uid)
	if err != nil {
		log.Errorf("Error getting UUID/Name for Container")
		return cni, err
	}
	log.Infof("updating cni with uuid %s name %s", containerUID, containerName)
	cni.Update(containerName, containerUID, "")
	cni.Log()
	return cni, nil

}

// Add implements contrailcno.ContrailCNIServer
func (s *server) Add(ctx context.Context, in *pb.CNIArgs) (*pb.AddResult, error) {
	/*
		skelArgs := getSkelArgs(in)
		addResult := &pb.AddResult{}
		cni, err := contrailCni.Init(skelArgs)
		if err != nil {
			return addResult, err
		}

		argsList := strings.Split(skelArgs.Args, ";")
		var argMap = make(map[string]string)
		for _, arg := range argsList {
			argKV := strings.Split(arg, "=")
			argMap[argKV[0]] = argKV[1]
		}
		containerName := argMap["K8S_POD_NAME"]
		log.Info("containerName %s", containerName)
		uid, err := getPodIDFromKubeAPI(containerName, argMap["K8S_POD_NAMESPACE"])
		if err != nil {
			log.Error("couldn't get uid from kube %s\n", err)
		}
		containerUID := fmt.Sprintf("%s", uid)
		if err != nil {
			log.Errorf("Error getting UUID/Name for Container")
			return addResult, err
		}
		log.Infof("updating cni with uuid %s name %s", containerUID, containerName)
		cni.Update(containerName, containerUID, "")
		cni.Log()
	*/
	addResult := &pb.AddResult{}
	cni, err := getCNI(ctx, in)
	if err != nil {
		log.Errorf("Failed getting cni.")
		return addResult, err
	}
	log.Infof("Came in Add for cni.ContainerUuid %s", cni.ContainerUuid)
	// Handle Add command
	result, cniVersion, err := cni.CmdAdd()
	if err != nil {
		log.Errorf("Failed processing Add command.")
		return addResult, err
	}
	addResult = resultToProto(result, cniVersion)
	return addResult, nil
}

//Del implements contrailcno.ContrailCNIServer
func (s *server) Del(ctx context.Context, in *pb.CNIArgs) (*pb.DelResult, error) {
	delResult := &pb.DelResult{}
	cni, err := getCNI(ctx, in)
	if err != nil {
		log.Errorf("Failed getting cni.")
		return delResult, err
	}
	log.Infof("Came in Add for cni.ContainerUuid %s", cni.ContainerUuid)
	// Handle Del command
	err = cni.CmdDel()
	if err != nil {
		log.Errorf("Failed processing Add command.")
		return delResult, err
	}
	return delResult, nil
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Info("Received: %v", in.GetName())
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func resultToProto(result *current.Result, cniVersion string) *pb.AddResult {

	var protoInfList []*pb.Intf
	for _, intf := range result.Interfaces {
		protoIntf := &pb.Intf{
			Name:    intf.Name,
			Mac:     intf.Mac,
			Sandbox: intf.Sandbox,
		}
		protoInfList = append(protoInfList, protoIntf)
	}

	var protoIPList []*pb.IPConfig
	for _, ip := range result.IPs {
		intf := *ip.Interface
		protoIP := &pb.IPConfig{
			Version: ip.Version,
			Intf:    int32(intf),
			Address: ip.Address.String(),
			Gateway: ip.Gateway.String(),
		}
		protoIPList = append(protoIPList, protoIP)
	}

	var protoRouteList []*pb.Route
	for _, route := range result.Routes {
		protoRoute := &pb.Route{
			Dst: route.Dst.String(),
			GW:  route.GW.String(),
		}
		protoRouteList = append(protoRouteList, protoRoute)
	}

	protoDNS := &pb.DNS{
		Nameservers: result.DNS.Nameservers,
		Domain:      result.DNS.Domain,
		Search:      result.DNS.Search,
		Options:     result.DNS.Options,
	}

	addResult := &pb.AddResult{
		CNIVersion: result.CNIVersion,
		Interfaces: protoInfList,
		IPs:        protoIPList,
		Routes:     protoRouteList,
		DNSs:       protoDNS,
	}
	return addResult
}

func main() {
	fmt.Println("Serving...")
	log.Init("/var/log/contrail/cni/server.log", 10, 5)
	log.Info("Started serving")
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Error("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterContrailCNIServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Error("failed to serve: %v", err)
	}
}

func getPodIDFromKubeAPI(podName string, podNamespace string) (types.UID, error) {
	log.Info("getPodIDFromKubeAPI\n")
	var uid types.UID
	kubeconfig := "/etc/rancher/k3s/k3s.yaml"
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Error("cannot load kube config\n")
		return uid, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error("cannot create clientset\n")
		return uid, err
	}
	pod, err := clientset.CoreV1().Pods(podNamespace).Get(podName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Error("Pod %s in namespace %s not found\n", podName, podNamespace)
		return uid, err
	} else if err != nil {
		log.Error("Error getting pod %s in namespace %s\n", podName, podNamespace)
		return uid, err
	}

	log.Info("Found pod %s in namespace %s\n", podName, podNamespace)

	uid = pod.GetUID()

	return uid, nil
}

func getSkelArgs(cniArgs *pb.CNIArgs) *skel.CmdArgs {
	return &skel.CmdArgs{
		ContainerID: cniArgs.ContainerID,
		Netns:       cniArgs.Netns,
		IfName:      cniArgs.IfName,
		Args:        cniArgs.Args,
		Path:        cniArgs.Path,
		StdinData:   []byte(cniArgs.StdinData),
	}
}
