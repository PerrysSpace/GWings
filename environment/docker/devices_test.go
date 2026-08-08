package docker

import (
	"testing"

	"github.com/pelican-dev/wings/config"
)

func TestResolveDevices_NoAllowlist(t *testing.T) {
	resolved := resolveDevices(nil, []string{"ENABLE_GPU=1"})
	if !resolved.Empty() {
		t.Fatalf("expected no devices when allowlist is empty, got %+v", resolved)
	}
}

func TestResolveDevices_OptedIn(t *testing.T) {
	allowed := map[string]config.DeviceConfiguration{
		"gpu": {
			Paths:  []string{"/dev/dri"},
			Groups: []string{"video", "render"},
		},
	}

	resolved := resolveDevices(allowed, []string{"ENABLE_GPU=1"})

	if resolved.Empty() {
		t.Fatal("expected devices to be resolved")
	}
	if len(resolved.Devices) != 1 || resolved.Devices[0].PathOnHost != "/dev/dri" {
		t.Fatalf("expected /dev/dri to be mapped, got %+v", resolved.Devices)
	}
	if len(resolved.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %+v", resolved.Groups)
	}
}

func TestResolveDevices_NotOptedIn(t *testing.T) {
	allowed := map[string]config.DeviceConfiguration{
		"gpu": {Paths: []string{"/dev/dri"}},
	}

	resolved := resolveDevices(allowed, []string{"ENABLE_GPU=0"})

	if !resolved.Empty() {
		t.Fatalf("expected no devices when not opted in, got %+v", resolved)
	}
}

// The security-critical case: an env var cannot smuggle in a path for a group
// that was never defined in the allowlist.
func TestResolveDevices_UnknownGroupIgnored(t *testing.T) {
	allowed := map[string]config.DeviceConfiguration{
		"gpu": {Paths: []string{"/dev/dri"}},
	}

	resolved := resolveDevices(allowed, []string{"ENABLE_DOCKER_SOCKET=1"})

	if !resolved.Empty() {
		t.Fatalf("expected unknown group to be ignored, got %+v", resolved)
	}
}
