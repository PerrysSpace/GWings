package docker

import (
	"os"
	"path/filepath"
	"testing"
)

// writeCard builds a fake /sys/class/drm entry. Files are only created for the
// keys present in the map, so a test can describe a card that lacks a counter by
// leaving it out.
func writeCard(t *testing.T, root string, name string, files map[string]string) {
	t.Helper()

	for path, content := range files {
		full := filepath.Join(root, name, "device", path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("creating %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", full, err)
		}
	}
}

func TestReadGPUStats(t *testing.T) {
	t.Run("full amdgpu card", func(t *testing.T) {
		root := t.TempDir()
		writeCard(t, root, "card0", map[string]string{
			"gpu_busy_percent":            "37\n",
			"mem_info_vram_used":          "3980000000\n",
			"mem_info_vram_total":         "8589934592\n",
			"uevent":                      "DRIVER=amdgpu\nPCI_ID=1002:67DF\nPCI_SLOT_NAME=0000:01:00.0\n",
			"hwmon/hwmon2/temp1_input":    "72000\n",
			"hwmon/hwmon2/power1_average": "118500000\n",
		})

		stats := readGPUStats(root)
		if stats == nil {
			t.Fatal("expected stats, got nil")
		}
		if stats.Utilization != 37 {
			t.Errorf("utilization = %v, want 37", stats.Utilization)
		}
		if stats.Memory != 3980000000 {
			t.Errorf("memory = %v, want 3980000000", stats.Memory)
		}
		if stats.MemoryLimit != 8589934592 {
			t.Errorf("memory limit = %v, want 8589934592", stats.MemoryLimit)
		}
		if stats.Driver != "amdgpu" {
			t.Errorf("driver = %q, want amdgpu", stats.Driver)
		}
		// The kernel writes the PCI ID uppercase, it is normalised on the way in.
		if stats.PCIID != "1002:67df" {
			t.Errorf("pci id = %q, want 1002:67df", stats.PCIID)
		}
		if stats.Temperature != 72 {
			t.Errorf("temperature = %v, want 72", stats.Temperature)
		}
		if stats.Power != 118.5 {
			t.Errorf("power = %v, want 118.5", stats.Power)
		}
	})

	t.Run("no sensors is not an error", func(t *testing.T) {
		root := t.TempDir()
		writeCard(t, root, "card0", map[string]string{
			"gpu_busy_percent": "0\n",
		})

		stats := readGPUStats(root)
		if stats == nil {
			t.Fatal("expected stats, got nil")
		}
		if stats.Utilization != 0 {
			t.Errorf("utilization = %v, want 0", stats.Utilization)
		}
		if stats.Temperature != 0 || stats.Power != 0 || stats.MemoryLimit != 0 {
			t.Errorf("expected unset sensors to stay zero, got %+v", stats)
		}
	})

	t.Run("power1_input as a fallback", func(t *testing.T) {
		root := t.TempDir()
		writeCard(t, root, "card0", map[string]string{
			"gpu_busy_percent":          "5\n",
			"hwmon/hwmon0/power1_input": "42000000\n",
		})

		stats := readGPUStats(root)
		if stats == nil {
			t.Fatal("expected stats, got nil")
		}
		if stats.Power != 42 {
			t.Errorf("power = %v, want 42", stats.Power)
		}
	})

	t.Run("a card reporting zero power is treated as not reporting", func(t *testing.T) {
		root := t.TempDir()
		writeCard(t, root, "card0", map[string]string{
			"gpu_busy_percent":            "5\n",
			"hwmon/hwmon0/power1_average": "0\n",
			"hwmon/hwmon0/power1_input":   "31000000\n",
		})

		stats := readGPUStats(root)
		if stats == nil {
			t.Fatal("expected stats, got nil")
		}
		if stats.Power != 31 {
			t.Errorf("power = %v, want 31", stats.Power)
		}
	})

	t.Run("skips cards without a utilization counter", func(t *testing.T) {
		root := t.TempDir()
		writeCard(t, root, "card0", map[string]string{
			"uevent": "DRIVER=i915\n",
		})
		writeCard(t, root, "card1", map[string]string{
			"gpu_busy_percent": "12\n",
			"uevent":           "DRIVER=amdgpu\n",
		})

		stats := readGPUStats(root)
		if stats == nil {
			t.Fatal("expected stats, got nil")
		}
		if stats.Driver != "amdgpu" {
			t.Errorf("driver = %q, want amdgpu", stats.Driver)
		}
	})

	t.Run("no card at all", func(t *testing.T) {
		if stats := readGPUStats(t.TempDir()); stats != nil {
			t.Errorf("expected nil, got %+v", stats)
		}
	})

	t.Run("missing sysfs root", func(t *testing.T) {
		if stats := readGPUStats(filepath.Join(t.TempDir(), "nope")); stats != nil {
			t.Errorf("expected nil, got %+v", stats)
		}
	})

	t.Run("malformed counter is ignored rather than parsed as zero", func(t *testing.T) {
		root := t.TempDir()
		writeCard(t, root, "card0", map[string]string{
			"gpu_busy_percent": "N/A\n",
		})
		writeCard(t, root, "card1", map[string]string{
			"gpu_busy_percent": "88\n",
		})

		stats := readGPUStats(root)
		if stats == nil {
			t.Fatal("expected stats, got nil")
		}
		if stats.Utilization != 88 {
			t.Errorf("utilization = %v, want 88", stats.Utilization)
		}
	})
}

func TestGPUCardDirs(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"card0", "card1", "card10", "card2"} {
		writeCard(t, root, name, map[string]string{"uevent": "DRIVER=amdgpu\n"})
	}
	// Connector directories live next to the cards and must not be mistaken for
	// one, or the first "card" found would be a display output.
	writeCard(t, root, "card0-DP-1", map[string]string{"uevent": "\n"})
	writeCard(t, root, "renderD128", map[string]string{"uevent": "\n"})

	dirs := gpuCardDirs(root)
	want := []string{"card0", "card1", "card2", "card10"}

	if len(dirs) != len(want) {
		t.Fatalf("got %d dirs (%v), want %d", len(dirs), dirs, len(want))
	}
	for i, name := range want {
		if expected := filepath.Join(root, name, "device"); dirs[i] != expected {
			t.Errorf("dirs[%d] = %q, want %q", i, dirs[i], expected)
		}
	}
}

func TestReadUevent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "uevent")

	if err := os.WriteFile(path, []byte("DRIVER=amdgpu\nMODALIAS=pci:v1002\nPCI_ID=1002:67DF\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	driver, pciID := readUevent(path)
	if driver != "amdgpu" || pciID != "1002:67df" {
		t.Errorf("got (%q, %q), want (amdgpu, 1002:67df)", driver, pciID)
	}

	if driver, pciID := readUevent(filepath.Join(dir, "missing")); driver != "" || pciID != "" {
		t.Errorf("got (%q, %q) for a missing file, want empty", driver, pciID)
	}
}
