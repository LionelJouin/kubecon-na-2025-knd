package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/containerd/nri/pkg/stub"
	"github.com/lioneljouin/kubecon-na-2025-knd/pkg/driver"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	ctx := context.Background()

	driverName := "kubecon-na-2025-knd.example.com"
	pluginName := "kubecon-na-2025-knd"
	pluginIndex := "25"
	nodeName := os.Getenv("NODE_NAME")

	// Create Kubernetes client.
	clientCfg, err := rest.InClusterConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to InClusterConfig: %v", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(clientCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to NewForConfig: %v", err)
		os.Exit(1)
	}

	// 1. Create the in-memory store of ResourceClaims of pods.
	memoryStore := driver.NewMemoryStore()

	// 2. Start the DRA Kubelet plugin (DRA Driver).
	draDriver, err := driver.Start(
		ctx,
		driverName,
		nodeName,
		clientset,
		memoryStore,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to dra.Start: %v", err)
		os.Exit(1)
	}
	defer draDriver.Stop()

	// 3. Start the resource discovery.
	resourceDiscovery := &driver.Resources{
		PublishResourcesFunc: draDriver.PublishResources,
		Interval:             5 * time.Second,
		NodeName:             nodeName,
	}
	go resourceDiscovery.Run(ctx)

	// 4. Start the NRI plugin.
	p := &driver.Plugin{
		PodResourceStore: memoryStore,
		DriverName:       driverName,
		ClientSet:        clientset,
	}

	p.Stub, err = stub.New(p, []stub.Option{
		stub.WithPluginName(pluginName),
		stub.WithPluginIdx(pluginIndex),
	}...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create plugin stub: %v", err)
		os.Exit(1)
	}

	err = p.Stub.Run(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "plugin exited with error: %v", err)
		os.Exit(1)
	}

}
