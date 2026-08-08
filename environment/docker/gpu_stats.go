package docker

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pelican-dev/wings/config"
	"github.com/pelican-dev/wings/environment"
)

// gpuSysfsRoot is the directory the DRM cards are enumerated from. It is a
// variable rather than a constant so the tests can point it at a fixture tree.
var gpuSysfsRoot = "/sys/class/drm"

// gpuCardDirs returns the device directories of the DRM cards under root, in
// card index order. The connector directories the kernel also puts there
// (card0-DP-1 and friends) are skipped.
func gpuCardDirs(root string) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}

	type card struct {
		index int
		dir   string
	}

	var cards []card
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "card") {
			continue
		}
		index, err := strconv.Atoi(strings.TrimPrefix(name, "card"))
		if err != nil {
			// A connector such as "card0-DP-1", not a card.
			continue
		}
		cards = append(cards, card{index: index, dir: filepath.Join(root, name, "device")})
	}

	// ReadDir sorts lexically, which puts card10 before card2.
	sort.Slice(cards, func(i, j int) bool { return cards[i].index < cards[j].index })

	dirs := make([]string, 0, len(cards))
	for _, c := range cards {
		dirs = append(dirs, c.dir)
	}
	return dirs
}

// readUintFile reads a sysfs file holding a single unsigned number. The second
// return value is false whenever the file is missing, unreadable or does not
// hold a number, all of which are normal for sensors a given card lacks.
func readUintFile(path string) (uint64, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}

	v, err := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		return 0, false
	}

	return v, true
}

// readUevent pulls the driver name and PCI ID out of a device's uevent file. A
// missing or unreadable file just leaves both empty.
func readUevent(path string) (driver string, pciID string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}

	for _, line := range strings.Split(string(b), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "DRIVER":
			driver = strings.TrimSpace(v)
		case "PCI_ID":
			pciID = strings.ToLower(strings.TrimSpace(v))
		}
	}

	return driver, pciID
}

// readGPUStats reads the first DRM card under root that reports a utilization
// counter. Only the amdgpu driver is supported: it exposes everything needed as
// plain files in sysfs, which needs no library, no subprocess and no privileges
// beyond the ones Wings already has.
//
// Nothing here is per-process. Attributing load to a single container means
// walking the DRM fdinfo of the container's processes (what nvtop does), which
// is a good deal more machinery than this, and is the reason the values are
// documented as card-level throughout.
func readGPUStats(root string) *environment.GPUStats {
	for _, dir := range gpuCardDirs(root) {
		busy, ok := readUintFile(filepath.Join(dir, "gpu_busy_percent"))
		if !ok {
			// Either not an amdgpu card or a kernel too old to report it. Both
			// mean there is nothing to show, so try the next card.
			continue
		}

		stats := &environment.GPUStats{Utilization: float64(busy)}
		stats.Driver, stats.PCIID = readUevent(filepath.Join(dir, "uevent"))

		if used, ok := readUintFile(filepath.Join(dir, "mem_info_vram_used")); ok {
			stats.Memory = used
		}
		if total, ok := readUintFile(filepath.Join(dir, "mem_info_vram_total")); ok {
			stats.MemoryLimit = total
		}

		readHwmon(dir, stats)

		return stats
	}

	return nil
}

// readHwmon fills in the temperature and power readings from the card's hwmon
// device. Both are optional: which sensors exist depends on the card, and the
// zero value means "not reported" for each of them.
func readHwmon(dir string, stats *environment.GPUStats) {
	matches, err := filepath.Glob(filepath.Join(dir, "hwmon", "hwmon*"))
	if err != nil {
		return
	}
	sort.Strings(matches)

	for _, hwmon := range matches {
		// Millidegrees Celsius.
		if temp, ok := readUintFile(filepath.Join(hwmon, "temp1_input")); ok && stats.Temperature == 0 {
			stats.Temperature = float64(temp) / 1000
		}

		// Microwatts. Cards report either an average or an instantaneous value,
		// and some report neither unless they are under load.
		if stats.Power == 0 {
			for _, name := range []string{"power1_average", "power1_input"} {
				if power, ok := readUintFile(filepath.Join(hwmon, name)); ok && power > 0 {
					stats.Power = float64(power) / 1000000
					break
				}
			}
		}
	}
}

// gpuStats returns the state of the host GPU this server has access to, or nil
// if it has none or the card exposes no usable counters.
//
// Whether the server has a GPU is derived from the same allowlist the
// passthrough itself uses, so a server that did not opt into a device group
// never sees the node's card in its stats. resolveDevices walks a handful of map
// entries and the server's environment, which is cheap enough to redo on every
// stats tick and keeps this correct when the node's configuration changes.
func (e *Environment) gpuStats() *environment.GPUStats {
	if resolveDevices(config.Get().Docker.Devices, e.Configuration.EnvironmentVariables()).Empty() {
		return nil
	}

	return readGPUStats(gpuSysfsRoot)
}
