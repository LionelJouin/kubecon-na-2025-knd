package driver

import (
	"context"
	"fmt"
	"time"

	"github.com/vishvananda/netlink"
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
	// PublishResourcesFunc is the function to call to publish discovered resources
	// via the DRA driver.
	PublishResourcesFunc PublishResources
	// Interval is the duration between resource discovery attempts.
	Interval time.Duration
	// NodeName is the name of the node this resource discovery is running on.
	// It is used to set the NodeName of the discovered devices and as pool name.
	NodeName string
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

	links, err := netlink.LinkList()
	if err != nil {
		return nil, fmt.Errorf("failed to list links: %v", err)
	}

	for _, link := range links {
		mtu := int64(link.Attrs().MTU)
		linkType := link.Type()

		device := resourcev1.Device{
			Name: link.Attrs().Name,
			Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
				"kubecon-na-2025-knd.example.com/interfaceName": {StringValue: &link.Attrs().Name},
				"kubecon-na-2025-knd.example.com/type":          {StringValue: &linkType},
				"kubecon-na-2025-knd.example.com/mtu":           {IntValue: &mtu},
			},
			NodeName: &r.NodeName,
		}

		macAddress := link.Attrs().HardwareAddr.String()
		if macAddress != "" {
			device.Attributes["kubecon-na-2025-knd.example.com/macAddress"] = resourcev1.DeviceAttribute{StringValue: &macAddress}
		}

		pciAddress := link.Attrs().ParentDevBus
		if pciAddress != "" {
			device.Attributes[deviceattribute.StandardDeviceAttributePCIeRoot] = resourcev1.DeviceAttribute{StringValue: &pciAddress}
		}

		devices = append(devices, device)
	}

	return devices, nil
}
