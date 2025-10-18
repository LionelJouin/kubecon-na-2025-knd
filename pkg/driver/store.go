package driver

import (
	"sync"

	resourcev1 "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/types"
)

// MemoryStore implements an in-memory store of ResourceClaims of pods.
type MemoryStore struct {
	mu sync.RWMutex
	// podResources stores the ResourceClaims of pods.
	// The key is the pod UID.
	// The value is a slice of ResourceClaims associated with the pod.
	podResources map[types.UID][]*resourcev1.ResourceClaim
}

// NewMemoryStore returns a new in-memory store of ResourceClaims of pods.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		podResources: map[types.UID][]*resourcev1.ResourceClaim{},
	}
}

// Add stores the given ResourceClaim for the given pod UID.
func (m *MemoryStore) Add(podUID types.UID, claim *resourcev1.ResourceClaim) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Avoid claims to be stored twice.
	storedClaims, exists := m.podResources[podUID]
	if exists {
		for _, storedClaim := range storedClaims {
			if claim.UID == storedClaim.UID { // already stored.
				return
			}
		}
	}

	m.podResources[podUID] = append(m.podResources[podUID], claim)
}

// Get returns the ResourceClaims stored for the given pod UID.
func (m *MemoryStore) Get(podUID types.UID) []*resourcev1.ResourceClaim {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.podResources[podUID]
}

// Delete removes all ResourceClaims stored for the given pod UID.
func (m *MemoryStore) Delete(podUID types.UID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.podResources, podUID)
}
