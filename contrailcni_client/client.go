package main

import (
	"context"
	"fmt"
	"log"
	"time"

	cniSpecVersion "github.com/containernetworking/cni/pkg/version"
	pb "github.com/michaelhenkel/contrail-cni/contrailcni"

	"github.com/containernetworking/cni/pkg/skel"
	"google.golang.org/grpc"
)

const (
	address     = "localhost:10000"
	defaultName = "world"
)

func main() {
	log.Println("Contrail CNI client")
	skel.PluginMain(CmdAdd, CmdCheck, CmdDel, cniSpecVersion.All, "contrail")
	log.Println("Contrail CNI client")
}

//CmdAdd command calls the Add function
func CmdAdd(skelArgs *skel.CmdArgs) error {
	log.Println("Contrail CNI client")
	log.Printf("Calling binary with %v\n", skelArgs)
	c, ctx := newClient()
	addResult, err := c.Add(ctx, cniArgs(skelArgs))
	fmt.Printf("%s", addResult)
	if err != nil {
		log.Fatalf("could not add: %v", err)
	}

	return nil
}

//CmdCheck command calls the Check function
func CmdCheck(skelArgs *skel.CmdArgs) error {
	log.Println("Contrail command check")
	return nil
}

//CmdDel command calls the Del function
func CmdDel(skelArgs *skel.CmdArgs) error {
	log.Println("Contrail command delete")
	return nil
}

func newClient() (pb.ContrailCNIClient, context.Context) {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewContrailCNIClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return c, ctx
}

func cniArgs(skelArgs *skel.CmdArgs) *pb.CNIArgs {
	return &pb.CNIArgs{
		ContainerID: skelArgs.ContainerID,
		Netns:       skelArgs.Netns,
		IfName:      skelArgs.IfName,
		Args:        skelArgs.Args,
		Path:        skelArgs.Path,
		StdinData:   skelArgs.StdinData,
	}
}
