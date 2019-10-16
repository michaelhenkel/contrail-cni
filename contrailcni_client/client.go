package main

import (
	"context"
	"net"
	"strconv"
	"strings"
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
	log.Init("/var/log/contrail/cni/client.log", 10, 5)
	skel.PluginMain(CmdAdd, CmdCheck, CmdDel, cniSpecVersion.All, "contrail")
}

//CmdAdd command calls the Add function
func CmdAdd(skelArgs *skel.CmdArgs) error {

	c, ctx, conn, cancel := newClient()
	defer conn.Close()
	defer cancel()

	addResult, err := c.Add(ctx, cniArgs(skelArgs))
	if err != nil {
		log.Error("could not add: %v", err)
	}
	types.PrintResult(cniResult(addResult), addResult.GetCNIVersion())

	return nil
}

//CmdCheck command calls the Check function
func CmdCheck(skelArgs *skel.CmdArgs) error {

	return nil
}

//CmdDel command calls the Del function
func CmdDel(skelArgs *skel.CmdArgs) error {
	c, ctx, conn, cancel := newClient()
	defer conn.Close()
	defer cancel()

	_, err := c.Del(ctx, cniArgs(skelArgs))
	if err != nil {
		log.Error("could not add: %v", err)
	}
	return nil
}

func newClient() (pb.ContrailCNIClient, context.Context, *grpc.ClientConn, context.CancelFunc) {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Error("did not connect: %v", err)
	}
	c := pb.NewContrailCNIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	return c, ctx, conn, cancel
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
		addressSlice := strings.Split(ip.GetAddress(), "/")
		prefix := addressSlice[0]
		plen, err := strconv.Atoi(addressSlice[1])
		if err != nil {
			log.Error("couldn't convert plen: %v", err)
		}
		mask := net.CIDRMask(plen, 32)
		address := net.IPNet{IP: net.ParseIP(prefix), Mask: mask}
		gateway := net.ParseIP(ip.GetGateway())

		resultIP := &current.IPConfig{
			Version:   ip.GetVersion(),
			Interface: &intf,
			Address:   address,
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
		gateway := net.ParseIP(route.GetGW())
		if err != nil {
			log.Error("couldn't parse gateway: %v", err)
		}
		resultRoute := &types.Route{
			Dst: *dst,
			GW:  gateway,
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
