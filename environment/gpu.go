package environment

// GPUStats describes the state of a host GPU that a server has been granted
// access to through the device passthrough in environment/docker/devices.go.
//
// These numbers describe the *card*, not the server. Passing /dev/dri through to
// a container does not partition the GPU in any way, so two servers that opted
// into the same device group are both looking at the combined load of the card
// they share. Anything presenting these values has to say so.
type GPUStats struct {
	// The kernel driver backing the card, e.g. "amdgpu". Empty if it could not
	// be determined.
	Driver string `json:"driver,omitempty"`

	// The PCI vendor and device ID of the card in "1002:67df" form, taken from
	// the kernel rather than from a PCI ID database — this is not a marketing
	// name and is not meant to be shown as one.
	PCIID string `json:"pci_id,omitempty"`

	// Percentage of time the graphics engine was busy since the previous
	// reading, as reported by the driver.
	Utilization float64 `json:"utilization"`

	// Video memory currently in use on the card, in bytes, across every process
	// using it.
	Memory uint64 `json:"memory_bytes"`

	// Total video memory on the card, in bytes.
	MemoryLimit uint64 `json:"memory_limit_bytes"`

	// Temperature of the card in degrees Celsius. Omitted when the driver does
	// not expose a sensor.
	Temperature float64 `json:"temperature_c,omitempty"`

	// Power draw of the card in watts. Omitted when the driver does not expose
	// it — several cards report power only while under load, and older ones not
	// at all.
	Power float64 `json:"power_watts,omitempty"`
}
