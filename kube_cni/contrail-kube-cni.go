// vim: tabstop=4 expandtab shiftwidth=4 softtabstop=4
//
// Copyright (c) 2017 Juniper Networks, Inc. All rights reserved.
//
/****************************************************************************
 * Main routines for kubernetes CNI plugin
 ****************************************************************************/
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	contrailCni "github.com/michaelhenkel/contrail-cni/contrail"

	//"github.com/michaelhenkel/contrail-cni/common"
	"github.com/containernetworking/cni/pkg/skel"
	cniSpecVersion "github.com/containernetworking/cni/pkg/version"
	log "github.com/michaelhenkel/contrail-cni/logging"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func getPodIDFromKubeAPI(podName string, podNamespace string) (types.UID, error) {
	log.Info("getPodIDFromKubeAPI\n")
	var kubeconfig *string
	var uid types.UID
	/*
		if home := homeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "/etc/rancher/k3s/k3s.yaml", "absolute path to the kubeconfig file")
		}
	*/
	kubeconfig = flag.String("kubeconfig", "/etc/rancher/k3s/k3s.yaml", "absolute path to the kubeconfig file")
	flag.Parse()
	log.Info("got flags %s\n", kubeconfig)
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Error("cannot load kube config\n")
		return uid, err
	}
	log.Info("got kubeconfig\n")
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error("cannot create clientset\n")
		return uid, err
	}
	log.Info("got clientset\n")
	pod, err := clientset.CoreV1().Pods(podNamespace).Get(podName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Error("Pod %s in namespace %s not found\n", podName, podNamespace)
		return uid, err
	} else if err != nil {
		log.Error("Error getting pod %s in namespace %s\n", podName, podNamespace)
		return uid, err
	}
	log.Info("got pod\n")

	log.Info("Found pod %s in namespace %s\n", podName, podNamespace)

	uid = pod.GetUID()

	return uid, nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

// Add command
func CmdAdd(skelArgs *skel.CmdArgs) error {
	// Initialize ContrailCni module

	cni, err := contrailCni.Init(skelArgs)
	if err != nil {
		return err
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
	log.Info("UID %s\n", containerUID)
	if err != nil {
		log.Errorf("Error getting UUID/Name for Container")
		return err
	}
	log.Infof("updating cni with uuid %s name %s", containerUID, containerName)
	cni.Update(containerName, containerUID, "", "")
	cni.Log()
	log.Infof("Came in Add for cni.ContainerUuid %s", cni.ContainerUuid)

	// Handle Add command
	_, _, err = cni.CmdAdd()
	if err != nil {
		log.Errorf("Failed processing Add command.")
		return err
	}

	return nil
}

// Del command
func CmdDel(skelArgs *skel.CmdArgs) error {
	// Initialize ContrailCni module
	cni, err := contrailCni.Init(skelArgs)
	if err != nil {
		return err
	}

	log.Infof("Came in Del for container %s", skelArgs.ContainerID)
	// Get UUID and Name for container
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

	// Update UUID and Name for container
	cni.Update(containerName, containerUID, "", "")
	cni.Log()

	// Handle Del command
	err = cni.CmdDel()
	if err != nil {
		log.Errorf("Failed processing Del command.")
		return err
	}

	return nil
}

// Check command
func CmdCheck(skelArgs *skel.CmdArgs) error {

	return nil
}

func main() {
	// Let CNI skeletal code handle demux based on env variables

	skel.PluginMain(CmdAdd, CmdCheck, CmdDel, cniSpecVersion.All, "contrail")
}
