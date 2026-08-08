package docker

import (
	"sort"
	"strings"

	"github.com/docker/docker/api/types/container"

	"github.com/pelican-dev/wings/config"
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

// resolveDevices determines which of the device groups configured by the node
// administrator a server has opted into, and returns the resulting device
// mappings and supplemental groups.
//
// The allowlist is the only source of device paths. A server opts in by setting
// `ENABLE_<NAME>` to a truthy value, where <NAME> is the name of a group defined
// in the Wings configuration — the value of that variable is never interpreted as
// a path, so a server is unable to request a device the administrator did not
// explicitly allow.
func resolveDevices(allowed map[string]config.DeviceConfiguration, envs []string) resolvedDevices {
	var resolved resolvedDevices
	if len(allowed) == 0 {
		return resolved
	}

	seenDevices := make(map[string]struct{})
	seenGroups := make(map[string]struct{})

	// Iterate over the allowlist rather than over the environment variables so
	// that a server can only ever match something that has been configured.
	names := make([]string, 0, len(allowed))
	for name := range allowed {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if !deviceGroupEnabled(name, envs) {
			continue
		}

		resolved.Names = append(resolved.Names, name)
		for _, p := range allowed[name].Paths {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if _, ok := seenDevices[p]; ok {
				continue
			}
			seenDevices[p] = struct{}{}
			resolved.Devices = append(resolved.Devices, container.DeviceMapping{
				PathOnHost:        p,
				PathInContainer:   p,
				CgroupPermissions: "rwm",
			})
		}

		for _, g := range allowed[name].Groups {
			g = strings.TrimSpace(g)
			if g == "" {
				continue
			}
			if _, ok := seenGroups[g]; ok {
				continue
			}
			seenGroups[g] = struct{}{}
			resolved.Groups = append(resolved.Groups, g)
		}
	}

	return resolved
}

// applyDevices grants the container access to the device groups the server has
// opted into, as far as they are allowed by the node's configuration. It is a
// no-op when no devices are configured, which is the default.
func (e *Environment) applyDevices(hostConf *container.HostConfig, envs []string) {
	resolved := resolveDevices(config.Get().Docker.Devices, envs)
	if resolved.Empty() {
		return
	}

	hostConf.Resources.Devices = append(hostConf.Resources.Devices, resolved.Devices...)
	hostConf.GroupAdd = append(hostConf.GroupAdd, resolved.Groups...)

	e.log().WithField("device_groups", resolved.Names).Debug("environment/docker: passing host devices through to container")
}
