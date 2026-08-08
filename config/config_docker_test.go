package config

import (
	"testing"

	"gopkg.in/yaml.v2"
)

func TestDockerDevicesRoundTrip(t *testing.T) {
	raw := `
devices:
  gpu:
    paths: ["/dev/dri"]
    groups: ["video", "render"]
`
	var cfg DockerConfiguration
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	gpu, ok := cfg.Devices["gpu"]
	if !ok {
		t.Fatal("expected \"gpu\" device group to be parsed")
	}
	if len(gpu.Paths) != 1 || gpu.Paths[0] != "/dev/dri" {
		t.Fatalf("unexpected paths: %+v", gpu.Paths)
	}
	if len(gpu.Groups) != 2 {
		t.Fatalf("unexpected groups: %+v", gpu.Groups)
	}

	// Round-trip: marshal back out and re-parse, simulating Wings' WriteToDisk
	// rewriting the whole config file.
	out, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var reparsed DockerConfiguration
	if err := yaml.Unmarshal(out, &reparsed); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if len(reparsed.Devices) != 1 {
		t.Fatalf("devices did not survive round-trip, got %+v", reparsed.Devices)
	}
}

func TestDockerRegistryCredentialsForImage(t *testing.T) {
	cfg := DockerConfiguration{
		Registries: map[string]RegistryConfiguration{
			"registry.example.com": {
				Username: "registry",
				Password: "secret",
			},
			"registry.example.com/team": {
				Username: "team",
				Password: "secret",
			},
			"registry.example.com:5000": {
				Username: "port",
				Password: "secret",
			},
			"https://index.docker.io/v1/": {
				Username: "docker",
				Password: "secret",
			},
		},
	}

	tests := []struct {
		name     string
		image    string
		username string
	}{
		{
			name:     "registry domain",
			image:    "registry.example.com/project/image:latest",
			username: "registry",
		},
		{
			name:     "registry with port",
			image:    "registry.example.com:5000/project/image:latest",
			username: "port",
		},
		{
			name:     "registry path",
			image:    "registry.example.com/team/image:latest",
			username: "team",
		},
		{
			name:  "registry prefix is not domain",
			image: "registry.example.com.evil/project/image:latest",
		},
		{
			name:     "registry path prefix falls back to domain",
			image:    "registry.example.com/team-evil/image:latest",
			username: "registry",
		},
		{
			name:     "legacy docker hub registry",
			image:    "docker.io/library/busybox:latest",
			username: "docker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, registry := cfg.RegistryCredentialsForImage(tt.image)
			if tt.username == "" {
				if registry != nil {
					t.Fatalf("expected no registry credentials, got username %q", registry.Username)
				}

				return
			}

			if registry == nil {
				t.Fatalf("expected registry credentials for %q", tt.image)
			}

			if registry.Username != tt.username {
				t.Fatalf("expected username %q, got %q", tt.username, registry.Username)
			}
		})
	}
}

func TestDockerRegistryPathCredentialsDoNotMatchSiblingPath(t *testing.T) {
	cfg := DockerConfiguration{
		Registries: map[string]RegistryConfiguration{
			"registry.example.com/team": {
				Username: "team",
				Password: "secret",
			},
		},
	}

	_, registry := cfg.RegistryCredentialsForImage("registry.example.com/team-evil/image:latest")
	if registry != nil {
		t.Fatalf("expected no registry credentials, got username %q", registry.Username)
	}
}
