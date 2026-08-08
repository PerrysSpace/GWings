package docker

import (
	"github.com/docker/docker/api/types/container"
)

// resolvedDevices holds everything that needs to be applied to a container's
// HostConfig in order to grant it access to the device groups it opted into.
type resolvedDevices struct {
	// Names of the device groups that were matched, sorted alphabetically.
	Names []string
	// Devices to map into the container.
	Devices []container.DeviceMapping
	// Supplemental groups the container process should be added to.
	Groups []string
}

// Empty returns true if no device group was matched.
func (r resolvedDevices) Empty() bool {
	return len(r.Devices) == 0 && len(r.Groups) == 0
}
