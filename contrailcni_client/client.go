package main

import (
	"context"
	"net"
	"time"

	cniSpecVersion "github.com/containernetworking/cni/pkg/version"
	pb "github.com/michaelhenkel/contrail-cni/contrailcni"
	log "github.com/michaelhenkel/contrail-cni/logging"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"google.golang.org/grpc"
)

const (
	address     = "localhost:10000"
	defaultName = "world"
)

func main() {
	//log.Println("Contrail CNI client")
	log.Init("/var/log/contrail/cni/client.log", 10, 5)
	skel.PluginMain(CmdAdd, CmdCheck, CmdDel, cniSpecVersion.All, "contrail")
	//log.Println("Contrail CNI client")
}

//CmdAdd command calls the Add function
func CmdAdd(skelArgs *skel.CmdArgs) error {
	//log.Println("Contrail CNI client")
	//log.Printf("Calling binary with %v\n", skelArgs)
	log.Info("skelArgs: %+v\n", skelArgs)
	log.Info("cniArgs(skelArgs): %+v\n", cniArgs(skelArgs))
	//c, ctx := newClient()
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Error("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewContrailCNIClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: "bla"})
	if err != nil {
		log.Error("could not greet: %v", err)
	}
	log.Info("Greeting: %s", r.GetMessage())

	addResult, err := c.Add(ctx, cniArgs(skelArgs))

	if err != nil {
		log.Error("could not add: %v", err)
	}
	//log.Info("%s", addResult)
	log.Info("HELLLLO")
	log.Info("%s", cniResult(addResult))
	types.PrintResult(cniResult(addResult), addResult.GetCNIVersion())

	//args := getSkelArgs(addResult.)

	return nil
}

//CmdCheck command calls the Check function
func CmdCheck(skelArgs *skel.CmdArgs) error {
	//log.Println("Contrail command check")
	return nil
}

//CmdDel command calls the Del function
func CmdDel(skelArgs *skel.CmdArgs) error {
	//log.Println("Contrail command delete")
	return nil
}

func newClient() (pb.ContrailCNIClient, context.Context) {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Error("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewContrailCNIClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return c, ctx
}

func cniResult(addResult *pb.AddResult) *current.Result {

	var intfList []*current.Interface
	for _, intf := range addResult.Interfaces {
		resultIntf := &current.Interface{
			Name:    intf.GetName(),
			Mac:     intf.GetMac(),
			Sandbox: intf.GetSandbox(),
		}
		intfList = append(intfList, resultIntf)
	}

	var IPList []*current.IPConfig
	for _, ip := range addResult.IPs {
		intf := int(ip.GetIntf())
		_, address, err := net.ParseCIDR(ip.GetAddress())
		if err != nil {
			log.Error("couldn't parse address: %v", err)
		}
		gateway, _, err := net.ParseCIDR(ip.GetGateway())
		if err != nil {
			log.Error("couldn't parse gateway: %v", err)
		}
		resultIP := &current.IPConfig{
			Version:   ip.GetVersion(),
			Interface: &intf,
			Address:   *address,
			Gateway:   gateway,
		}
		IPList = append(IPList, resultIP)
	}

	var routeList []*types.Route
	for _, route := range addResult.Routes {
		_, dst, err := net.ParseCIDR(route.GetDst())
		if err != nil {
			log.Error("couldn't parse dst: %v", err)
		}
		gw, _, err := net.ParseCIDR(route.GetGW())
		if err != nil {
			log.Error("couldn't parse gw: %v", err)
		}
		resultRoute := &types.Route{
			Dst: *dst,
			GW:  gw,
		}
		routeList = append(routeList, resultRoute)
	}

	resultDNS := &types.DNS{
		Nameservers: addResult.DNSs.GetNameservers(),
		Domain:      addResult.DNSs.GetDomain(),
		Search:      addResult.DNSs.GetSearch(),
		Options:     addResult.DNSs.GetOptions(),
	}

	return &current.Result{
		CNIVersion: addResult.GetCNIVersion(),
		Interfaces: intfList,
		IPs:        IPList,
		Routes:     routeList,
		DNS:        *resultDNS,
	}
}

func cniArgs(skelArgs *skel.CmdArgs) *pb.CNIArgs {
	return &pb.CNIArgs{
		ContainerID: skelArgs.ContainerID,
		Netns:       skelArgs.Netns,
		IfName:      skelArgs.IfName,
		Args:        skelArgs.Args,
		Path:        skelArgs.Path,
		StdinData:   string(skelArgs.StdinData),
	}
}
