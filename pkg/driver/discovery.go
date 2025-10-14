package driver

import (
	"context"
	"fmt"
	"time"

	"github.com/jaypipes/ghw"
	resourcev1 "k8s.io/api/resource/v1"
	"k8s.io/dynamic-resource-allocation/deviceattribute"
	"k8s.io/dynamic-resource-allocation/resourceslice"
	"k8s.io/klog/v2"
)

// PublishResources is a function type to advertise resources.
type PublishResources func(context.Context, resourceslice.DriverResources) error

// Resources periodically discovers network devices
// and publishes them via the PublishResourcesFunc.
type Resources struct {
	PublishResourcesFunc PublishResources
	Interval             time.Duration
	NodeName             string
}

// Run starts the periodic discovery of network devices
// and publishing them via the PublishResourcesFunc.
func (r *Resources) Run(ctx context.Context) {
	for {
		devices, err := r.ListDevices()
		if err != nil {
			klog.FromContext(ctx).Error(err, "failed to list devices")
			continue
		}

		klog.FromContext(ctx).Info("devices to publish", "devices", devices)

		driverResources := resourceslice.DriverResources{
			Pools: map[string]resourceslice.Pool{
				r.NodeName: {
					Slices: []resourceslice.Slice{
						{Devices: devices},
					},
				},
			},
		}

		err = r.PublishResourcesFunc(ctx, driverResources)
		if err != nil {
			klog.FromContext(ctx).Error(err, "failed to publish resources")
			continue
		}

		select {
		case <-time.After(r.Interval):
		case <-ctx.Done():
			return
		}
	}
}

// ListDevices lists the network devices available on the node using ghw library.
// It returns a slice of resourcev1.Device representing the network devices.
// The device attributes include the interface name, MAC address, and PCI address (if available).
func (r *Resources) ListDevices() ([]resourcev1.Device, error) {
	devices := []resourcev1.Device{}

	netInfo, err := ghw.Network()
	if err != nil {
		return nil, fmt.Errorf("failed to get network info: %v", err)
	}

	for _, nic := range netInfo.NICs {
		device := resourcev1.Device{
			Name: nic.Name,
			Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
				"kubecon-na-2025-knd.example.com/interfaceName": {StringValue: &nic.Name},
			},
			NodeName: &r.NodeName,
		}

		if nic.MACAddress != "" {
			device.Attributes["kubecon-na-2025-knd.example.com/macAddress"] = resourcev1.DeviceAttribute{StringValue: &nic.MACAddress}
		}

		if nic.PCIAddress != nil && *nic.PCIAddress != "" {
			device.Attributes[deviceattribute.StandardDeviceAttributePCIeRoot] = resourcev1.DeviceAttribute{StringValue: nic.PCIAddress}
		}

		devices = append(devices, device)
	}

	return devices, nil
}
