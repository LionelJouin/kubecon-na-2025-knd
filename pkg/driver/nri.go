package driver

import (
	"context"
	"fmt"

	"github.com/containerd/nri/pkg/api"
	"github.com/containerd/nri/pkg/stub"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	resourcev1 "k8s.io/api/resource/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// Plugin implements a NRI plugin that adjusts containers
// to add the network devices allocated to the pod.
type Plugin struct {
	Stub stub.Stub
	// PodResourceStore is used to store the ResourceClaims of pods.
	PodResourceStore *MemoryStore
	// DriverName is the name of the DRA driver.
	// It is used to identify the devices allocated by this driver and
	// avoid adding devices allocated by other drivers.
	DriverName string
	// ClientSet is the Kubernetes clientset used to update ResourceClaims status.
	ClientSet clientset.Interface
}

// CreateContainer is called by the container runtime via the NRI API
// when a new container is being created.
// It returns a ContainerAdjustment that adds the network devices
// allocated to the pod to the container.
func (p *Plugin) CreateContainer(ctx context.Context, pod *api.PodSandbox, ctr *api.Container) (*api.ContainerAdjustment, []*api.ContainerUpdate, error) {
	adjust := &api.ContainerAdjustment{}

	claims := p.PodResourceStore.Get(types.UID(pod.GetUid()))

	for _, claim := range claims {
		for _, result := range claim.Status.Allocation.Devices.Results {
			if result.Driver != p.DriverName {
				continue
			}

			adjust.AddLinuxNetDevice(result.Device, &api.LinuxNetDevice{Name: result.Device})
		}
	}

	klog.FromContext(ctx).Info("CreateContainer: LinuxNetDevice requested from the ResourceClaims added to the container", "pod.UID", pod.GetUid(), "adjust", adjust)

	return adjust, nil, nil
}

// StartContainer is called by the container runtime via the NRI API
// when a container is started.
// It updates the ResourceClaim status to include the network data
// of the devices allocated to the pod.
func (p *Plugin) StartContainer(ctx context.Context, pod *api.PodSandbox, ctr *api.Container) error {
	claims := p.PodResourceStore.Get(types.UID(pod.GetUid()))

	podNetworkNamespace := getNetworkNamespace(pod)
	if podNetworkNamespace == "" {
		err := fmt.Errorf("error getting network namespace for pod '%s' in namespace '%s'", pod.Name, pod.Namespace)
		klog.FromContext(ctx).Info(err.Error())
		return err
	}

	fmt.Println("Pod Network Namespace:", podNetworkNamespace)

	for _, claim := range claims {
		devices := []resourcev1.AllocatedDeviceStatus{}

		// Keep devices allocated by other drivers.
		for _, device := range claim.Status.Devices {
			if device.Driver != p.DriverName {
				devices = append(devices, device)
			}
		}

		// Add devices allocated by this driver.
		for _, result := range claim.Status.Allocation.Devices.Results {
			// Only consider devices allocated by this driver.
			if result.Driver != p.DriverName {
				continue
			}

			networkData, err := getNetworkData(result.Device, podNetworkNamespace)
			if err != nil {
				err = fmt.Errorf("failed to get network data for device '%s': %v", result.Device, err)
				klog.FromContext(ctx).Info(err.Error())
				return err
			}

			devices = append(devices, resourcev1.AllocatedDeviceStatus{
				Driver:  result.Driver,
				Pool:    result.Pool,
				Device:  result.Device,
				ShareID: (*string)(result.ShareID),
				Conditions: []v1.Condition{
					{
						Type:               "NetworkReady",
						Status:             v1.ConditionTrue,
						LastTransitionTime: v1.Now(),
						Reason:             "NetworkReady",
						Message:            "Device successfully allocated and assigned to the pod",
					},
				},
				NetworkData: networkData,
			})
		}

		// Update the ResourceClaim status with the new devices.
		claim.Status.Devices = devices
		_, err := p.ClientSet.ResourceV1().ResourceClaims(claim.GetNamespace()).UpdateStatus(ctx, claim, v1.UpdateOptions{})
		if err != nil {
			err = fmt.Errorf("failed to update resource claim status: %v", err)
			klog.FromContext(ctx).Info(err.Error())
			return err
		}
	}

	klog.FromContext(ctx).Info("PostCreateContainer: ResourceClaims Device Status updated", "pod.UID", pod.GetUid())

	return nil
}

// getNetworkNamespace returns the network namespace path of the pod.
func getNetworkNamespace(pod *api.PodSandbox) string {
	for _, namespace := range pod.Linux.GetNamespaces() {
		if namespace.Type == "network" {
			return namespace.Path
		}
	}

	return ""
}

// getNetworkData retrieves network data for the specified device
// in the given network namespace.
func getNetworkData(device string, networkNamespace string) (*resourcev1.NetworkDeviceData, error) {
	// Get a handle to the network namespace.
	handle, err := netns.GetFromPath(networkNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get network namespace handle: %v", err)
	}
	defer handle.Close()

	// Switch to the pod's network namespace.
	err = netns.Set(handle)
	if err != nil {
		return nil, fmt.Errorf("failed to set network namespace: %v", err)
	}

	// Get the network link from the device name.
	link, err := netlink.LinkByName(device)
	if err != nil {
		return nil, fmt.Errorf("failed to get link by name '%s': %v", device, err)
	}

	// Get the IP addresses assigned to the link.
	ips, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return nil, fmt.Errorf("failed to get IP addresses for link '%s': %v", link.Attrs().Name, err)
	}

	// Prepare the list of IP addresses as strings.
	ipList := make([]string, 0, len(ips))
	for _, ip := range ips {
		ipList = append(ipList, ip.IPNet.String())
	}

	return &resourcev1.NetworkDeviceData{
		InterfaceName:   link.Attrs().Name,
		IPs:             ipList,
		HardwareAddress: link.Attrs().HardwareAddr.String(),
	}, nil
}
