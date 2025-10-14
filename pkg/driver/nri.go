package driver

import (
	"context"

	"github.com/containerd/nri/pkg/api"
	"github.com/containerd/nri/pkg/stub"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

type Plugin struct {
	Stub             stub.Stub
	PodResourceStore *MemoryStore
	DriverName       string
}

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

	klog.FromContext(ctx).Info("CreateContainer", "pod.UID", pod.GetUid(), "adjust", adjust)

	return adjust, nil, nil
}
