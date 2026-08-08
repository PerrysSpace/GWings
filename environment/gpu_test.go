package environment

import (
	"testing"

	"github.com/goccy/go-json"
)

func TestGPUStatsJSONKeys(t *testing.T) {
	stats := GPUStats{
		Driver:      "amdgpu",
		PCIID:       "1002:67df",
		Utilization: 42.5,
		Memory:      1024,
		MemoryLimit: 8589934592,
		Temperature: 65.0,
		Power:       95.5,
	}

	b, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, key := range []string{"driver", "pci_id", "utilization", "memory_bytes", "memory_limit_bytes", "temperature_c", "power_watts"} {
		if _, ok := raw[key]; !ok {
			t.Fatalf("expected JSON key %q, got keys %v", key, raw)
		}
	}
}

func TestGPUStatsOmitsEmptyOptionalFields(t *testing.T) {
	stats := GPUStats{Utilization: 10, Memory: 512}

	b, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, key := range []string{"driver", "pci_id", "temperature_c", "power_watts"} {
		if _, ok := raw[key]; ok {
			t.Fatalf("expected %q to be omitted when empty", key)
		}
	}
}
