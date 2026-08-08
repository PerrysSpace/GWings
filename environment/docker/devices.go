package docker

import (
	"strings"

	"github.com/docker/docker/api/types/container"
	//"github.com/pelican-dev/wings/config"
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

// deviceOptInPrefix is prepended to the (normalized) name of a device group to
// build the environment variable a server uses to opt into that group.
const deviceOptInPrefix = "ENABLE_"

// deviceOptInVariable returns the environment variable used to opt into the
// device group with the given name, e.g. "gpu" becomes "ENABLE_GPU".
func deviceOptInVariable(name string) string {
	var b strings.Builder
	b.WriteString(deviceOptInPrefix)
	for _, r := range strings.ToUpper(strings.TrimSpace(name)) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

// deviceGroupEnabled returns true if the server's environment opts into the
// given device group.
func deviceGroupEnabled(name string, envs []string) bool {
	key := deviceOptInVariable(name)
	if key == deviceOptInPrefix {
		return false
	}

	// Walk the environment backwards, the last definition of a variable is the
	// one Docker ends up using.
	for i := len(envs) - 1; i >= 0; i-- {
		k, v, ok := strings.Cut(envs[i], "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(k), key) {
			continue
		}
		return isTruthy(v)
	}

	return false
}

// isTruthy returns true for the values that are accepted as "on" for a device
// opt-in variable. Anything else — including a path — disables the group.
func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}
