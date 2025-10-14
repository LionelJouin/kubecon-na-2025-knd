package driver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	resourcev1 "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/dynamic-resource-allocation/kubeletplugin"
	"k8s.io/dynamic-resource-allocation/resourceslice"
	"k8s.io/klog/v2"
)

var _ kubeletplugin.DRAPlugin = &Driver{}

// Driver represents a DRA Kubelet plugin.
type Driver struct {
	driverName       string
	kubeClient       kubernetes.Interface
	draPlugin        *kubeletplugin.Helper
	podResourceStore *MemoryStore
}

// Start the DRA Kubelet plugin.
func Start(
	ctx context.Context,
	driverName string,
	nodeName string,
	kubeClient kubernetes.Interface,
	podResourceStore *MemoryStore,
) (*Driver, error) {
	driver := &Driver{
		driverName:       driverName,
		kubeClient:       kubeClient,
		podResourceStore: podResourceStore,
	}

	driverPluginPath := filepath.Join("/var/lib/kubelet/plugins/", driverName)

	err := os.MkdirAll(driverPluginPath, 0750)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin path %s: %v", driverPluginPath, err)
	}

	plugin, err := kubeletplugin.Start(
		ctx,
		driver,
		kubeletplugin.KubeClient(kubeClient),
		kubeletplugin.NodeName(nodeName),
		kubeletplugin.DriverName(driverName),
	)

	if err != nil {
		return nil, fmt.Errorf("start kubelet plugin: %w", err)
	}

	driver.draPlugin = plugin

	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 30*time.Second, true, func(context.Context) (bool, error) {
		status := driver.draPlugin.RegistrationStatus()
		if status == nil {
			return false, nil
		}

		return status.PluginRegistered, nil
	})
	if err != nil {
		return nil, err
	}

	return driver, nil
}

// Stop the DRA Kubelet plugin.
func (d *Driver) Stop() {
	if d.draPlugin != nil {
		d.draPlugin.Stop()
	}
}

// PublishResources publishes the given resources via the DRA Kubelet plugin.
func (d *Driver) PublishResources(ctx context.Context, resources resourceslice.DriverResources) error {
	return d.draPlugin.PublishResources(ctx, resources)
}

// PrepareResourceClaims prepares the resource claims for the given pod by storing
// the claims in the podResourceStore and returning the devices allocated by this driver.
func (d *Driver) PrepareResourceClaims(ctx context.Context, claims []*resourcev1.ResourceClaim) (result map[types.UID]kubeletplugin.PrepareResult, err error) {
	result = make(map[types.UID]kubeletplugin.PrepareResult)
	for _, claim := range claims {
		devices, err := d.nodePrepareResource(ctx, claim)

		var claimResult kubeletplugin.PrepareResult
		if err != nil {
			klog.FromContext(ctx).Error(err, "error unpreparing ressources for a claim", "claim.Namespace", claim.Namespace, "claim.Name", claim.Name)
			claimResult.Err = err
		} else {
			claimResult.Devices = devices
		}

		result[claim.UID] = claimResult
	}
	return result, nil
}

// HandleError handles errors from the DRA Kubelet plugin.
func (d *Driver) HandleError(_ context.Context, _ error, _ string) {
	// todo
}

// UnprepareResourceClaims unprepares the resource claims.
func (d *Driver) UnprepareResourceClaims(ctx context.Context, claims []kubeletplugin.NamespacedObject) (result map[types.UID]error, err error) {
	result = make(map[types.UID]error)

	for _, claimRef := range claims {
		uid := string(claimRef.UID)
		err := d.nodeUnprepareResource(ctx, uid)
		result[claimRef.UID] = err
	}

	return result, nil
}

func (d *Driver) nodePrepareResource(ctx context.Context, claim *resourcev1.ResourceClaim) ([]kubeletplugin.Device, error) {
	d.podResourceStore.Add(claim.Status.ReservedFor[0].UID, claim)

	var devices []kubeletplugin.Device
	for _, result := range claim.Status.Allocation.Devices.Results {
		if result.Driver != d.driverName {
			continue
		}

		device := kubeletplugin.Device{
			Requests:   []string{result.Request},
			PoolName:   result.Pool,
			DeviceName: result.Device,
		}
		devices = append(devices, device)
	}

	klog.FromContext(ctx).Info("Devices for Claim", "claim.UID", claim.UID, "devices", devices)

	return devices, nil
}

func (d *Driver) nodeUnprepareResource(_ context.Context, _ string) error {
	// TODO
	return nil
}
