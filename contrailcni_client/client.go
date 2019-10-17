package main

import (
	"context"
	"encoding/json"
	"net"
	"os"
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

func main() {
	log.Init("/var/log/contrail/cni/client.log", 10, 5)
	log.Info("Client run")
	skel.PluginMain(CmdAdd, CmdCheck, CmdDel, cniSpecVersion.All, "contrail")
}

type CNIConfiguration struct {
	CNIVersion string      `json:"cniVersion"`
	Contrail   CNIContrail `json:"contrail"`
	Name       string      `json:"name"`
	Type       string      `json:"type"`
}

type CNIContrail struct {
	VrouterIP     string `json:"vrouter-ip"`
	VrouterPort   int    `json:"vrouter-port"`
	ConfigDir     string `json:"config-dir"`
	PollTimeout   int    `json:"poll-timeout"`
	PollRetries   int    `json:"poll-retries"`
	LogFile       string `json:"log-file"`
	LogLevel      string `json:"log-level"`
	CNISocketPath string `json:"cnisocket-path"`
}

//CmdAdd command calls the Add function
func CmdAdd(skelArgs *skel.CmdArgs) error {
	stdinData := string(skelArgs.StdinData)
	contrailCNIConfig := &CNIConfiguration{}
	err := json.Unmarshal([]byte(stdinData), contrailCNIConfig)
	if err != nil {
		log.Error("could not serialize json: %v", err)
	}
	c, ctx, conn, cancel := newClient(contrailCNIConfig.Contrail.CNISocketPath)
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
	stdinData := string(skelArgs.StdinData)
	contrailCNIConfig := &CNIConfiguration{}
	err := json.Unmarshal([]byte(stdinData), contrailCNIConfig)
	if err != nil {
		log.Error("could not serialize json: %v", err)
	}
	log.Info("stdinData %+v\n", contrailCNIConfig)
	c, ctx, conn, cancel := newClient(contrailCNIConfig.Contrail.CNISocketPath)
	defer conn.Close()
	defer cancel()

	_, err = c.Del(ctx, cniArgs(skelArgs))
	if err != nil {
		log.Error("could not add: %v", err)
	}
	return nil
}

func unixConnect(addr string, t time.Duration) (net.Conn, error) {
	unixAddr, err := net.ResolveUnixAddr("unix", addr)
	conn, err := net.DialUnix("unix", nil, unixAddr)
	return conn, err
}

func newClient(socket string) (pb.ContrailCNIClient, context.Context, *grpc.ClientConn, context.CancelFunc) {
	if socket == "" {
		socket = "/var/run/contrail/cni.socket"
	}
	if _, err := os.Stat(socket); os.IsNotExist(err) {
		log.Error("socket %s doesn't exist, server running?", socket)
	}
	log.Info("Dialing socket %s", socket)
	conn, err := grpc.Dial(socket, grpc.WithInsecure(), grpc.WithDialer(unixConnect))
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
