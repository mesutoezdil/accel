// Package device is the vendor-neutral model of an accelerator: a GPU, NPU,
// XPU, MLU, or any other card siltide can read.
package device

import (
	"sort"
	"strconv"
	"time"
)

// Vendor identifies who makes the accelerator.
type Vendor string

const (
	NVIDIA    Vendor = "nvidia"
	AMD       Vendor = "amd"
	Intel     Vendor = "intel"
	Apple     Vendor = "apple"
	Ascend    Vendor = "ascend"    // Huawei Ascend NPU
	Neuron    Vendor = "neuron"    // AWS Inferentia / Trainium
	Biren     Vendor = "biren"     // Biren GPU
	Cambricon Vendor = "cambricon" // Cambricon MLU
	Enflame   Vendor = "enflame"   // Enflame GCU
	Hygon     Vendor = "hygon"     // Hygon DCU
	Iluvatar  Vendor = "iluvatar"  // Iluvatar CoreX GPU
	Kunlunxin Vendor = "kunlunxin" // Kunlunxin XPU
	MetaX     Vendor = "metax"     // MetaX GPU
	MThreads  Vendor = "mthreads"  // Moore Threads GPU
	VastAI    Vendor = "vastai"    // VastAI accelerator
)

// Metric names one reading. A device carries only the metrics its vendor
// reports: a missing key is "not available", never zero.
type Metric string

const (
	Util           Metric = "util"            // percent busy
	MemUsed        Metric = "mem_used"        // bytes
	MemTotal       Metric = "mem_total"       // bytes
	MemBandwidth   Metric = "mem_bw"          // percent of memory bandwidth in use
	Temp           Metric = "temp"            // celsius
	Power          Metric = "power"           // watts
	PowerCap       Metric = "power_cap"       // watts
	ClockCore      Metric = "clock_core"      // MHz
	ClockMem       Metric = "clock_mem"       // MHz
	Fan            Metric = "fan"             // percent
	Throttle       Metric = "throttle"        // Throttle* bits
	EccCorrected   Metric = "ecc_corrected"   // error-correcting code (ECC) errors, lifetime count
	EccUncorrected Metric = "ecc_uncorrected" // lifetime count
	PCIeGen        Metric = "pcie_gen"        // current link generation
	PCIeWidth      Metric = "pcie_width"      // current link width (lanes)
	PCIeMaxGen     Metric = "pcie_max_gen"    // generation the slot and card support
	PCIeMaxWidth   Metric = "pcie_max_width"  // lanes the slot and card support
	PCIeRx         Metric = "pcie_rx"         // bytes per second into the device
	PCIeTx         Metric = "pcie_tx"         // bytes per second out of the device
	LinksActive    Metric = "links_active"    // high-speed interconnect links up (NVLink, HCCS, ...)
	LinksTotal     Metric = "links_total"     // links the device has
	Partitions     Metric = "partitions"      // hardware partitions in use, e.g. Multi-Instance GPU (MIG)
	Encoder        Metric = "encoder"         // percent video encoder busy
	Decoder        Metric = "decoder"         // percent video decoder busy
	Energy         Metric = "energy"          // joules since driver load
	PState         Metric = "pstate"          // performance state, 0 is fastest
	MemTemp        Metric = "mem_temp"        // celsius, memory
	TempSlowdown   Metric = "temp_slowdown"   // celsius at which clocks drop
	TempShutdown   Metric = "temp_shutdown"   // celsius at which the device stops
	RemappedRows   Metric = "remapped_rows"   // rows remapped after errors (lifetime)
	RemapPending   Metric = "remap_pending"   // 1 when a reset is needed to apply remaps
	RemapFailed    Metric = "remap_failed"    // 1 when a remap failed: replace the device
	RetiredPages   Metric = "retired_pages"   // pages retired after errors (lifetime)
	PCIeReplays    Metric = "pcie_replays"    // link replays (lifetime)
	ViolationPower Metric = "violation_power" // percent of time held below clocks by power
	ViolationTherm Metric = "violation_therm" // percent of time held below clocks by heat
	SMUtil         Metric = "sm_util"         // process: percent of streaming multiprocessor (SM) time
	NUMANode       Metric = "numa_node"       // non-uniform memory access (NUMA) node of the device's PCIe slot
	AERCorrected   Metric = "aer_corrected"   // PCIe Advanced Error Reporting (AER) correctable errors (lifetime)
	AERFatal       Metric = "aer_fatal"       // PCIe AER uncorrectable errors (lifetime)
	SMActive       Metric = "sm_active"       // percent of time a streaming multiprocessor (SM) was busy (profiling)
	SMOccupancy    Metric = "sm_occupancy"    // percent of warps resident (profiling)
	TensorActive   Metric = "tensor_active"   // percent of time tensor cores were busy (profiling)
	DRAMActive     Metric = "dram_active"     // percent of time memory was busy (profiling)
)

// Link is one high-speed interconnect link (NVLink, HCCS, XGMI, ...).
type Link struct {
	Index   int     `json:"index"`
	Active  bool    `json:"active"`
	Version string  `json:"version,omitempty"`
	Peer    string  `json:"peer,omitempty"` // bus address or device of the far end
	Rx      float64 `json:"rx,omitempty"`   // bytes per second
	Tx      float64 `json:"tx,omitempty"`   // bytes per second
	Errors  uint64  `json:"errors"`         // lifetime error counter sum
}

// Throttle bits: why the device runs below its maximum clock.
const (
	ThrottleIdle     = 1 << iota // nothing to do
	ThrottlePowerCap             // power limit or power brake
	ThrottleThermal              // temperature limit
	ThrottleHW                   // hardware slowdown
	ThrottleOther                // application clocks, sync boost, ...
)

// ThrottleNames lists the active reasons in bits.
func ThrottleNames(bits float64) []string {
	b := int(bits)
	var out []string
	for _, r := range []struct {
		bit  int
		name string
	}{{ThrottleIdle, "idle"}, {ThrottlePowerCap, "power cap"}, {ThrottleThermal, "thermal"}, {ThrottleHW, "hw slowdown"}, {ThrottleOther, "other"}} {
		if b&r.bit != 0 {
			out = append(out, r.name)
		}
	}
	return out
}

// Metrics holds the readings a source reported.
type Metrics map[Metric]float64

// Get returns a metric and whether it was reported.
func (m Metrics) Get(k Metric) (float64, bool) {
	v, ok := m[k]
	return v, ok
}

// Or returns the metric or def.
func (m Metrics) Or(k Metric, def float64) float64 {
	if v, ok := m[k]; ok {
		return v
	}
	return def
}

// Keys lists the reported metrics in a stable order.
func (m Metrics) Keys() []Metric {
	out := make([]Metric, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// State summarises what a device is doing.
type State string

const (
	StateDown   State = "down"   // reported no metrics
	StateIdle   State = "idle"   // below the idle threshold
	StateActive State = "active" // doing something
	StateBusy   State = "busy"   // near saturation
)

// Device is one accelerator with its current readings.
type Device struct {
	// ID is stable across runs: vendor plus UUID, serial, or bus address.
	// Devices from another node carry "node/" in front.
	ID      string    `json:"id"`
	Node    string    `json:"node,omitempty"`   // "" for the local machine
	Parent  string    `json:"parent,omitempty"` // ID of the physical device for a partition
	Vendor  Vendor    `json:"vendor"`
	Index   int       `json:"index"`
	Name    string    `json:"name"`
	Bus     string    `json:"bus,omitempty"`
	Metrics Metrics   `json:"metrics"`
	Procs   []Process `json:"procs,omitempty"`
	Links   []Link    `json:"links,omitempty"`
	// Allocated names who holds the device even without a running process
	// (Kubernetes pod-resources, Slurm job). "" when unknown.
	Allocated string `json:"allocated,omitempty"`
	// Source is where the metrics came from: nvml, ioreg, npu-smi, sysfs,
	// dcgmi, simulated. Derived values are marked in the UI, never here.
	Source string `json:"source,omitempty"`
	// Topology maps a peer device index to how the two are connected:
	// NV<n> (NVLink), PIX, PXB, PHB, NODE, SYS.
	Topology map[int]string `json:"topology,omitempty"`

	// Filled in by the collector, never by a provider.
	State       State    `json:"state,omitempty"`
	Health      int      `json:"health"` // 0-100, 100 is healthy
	HealthNotes []string `json:"health_notes,omitempty"`
	Outlier     bool     `json:"outlier,omitempty"`        // well below its siblings
	IdleAlloc   bool     `json:"idle_allocated,omitempty"` // has processes but sits idle
}

// New returns a device with an empty metric set. An empty key falls back to
// the index, which is only stable while the machine's inventory is.
func New(v Vendor, index int, name, key, bus string) Device {
	if key == "" {
		key = strconv.Itoa(index)
	}
	return Device{ID: string(v) + "-" + key, Vendor: v, Index: index, Name: name, Bus: bus, Metrics: Metrics{}}
}

// MemPercent is used memory as a share of total, when both are known.
func (d Device) MemPercent() (float64, bool) {
	used, ok1 := d.Metrics.Get(MemUsed)
	total, ok2 := d.Metrics.Get(MemTotal)
	if !ok1 || !ok2 || total <= 0 {
		return 0, false
	}
	return used / total * 100, true
}

// Label is "node:index" or "index" for tables.
func (d Device) Label() string {
	if d.Node != "" {
		return d.Node + ":" + strconv.Itoa(d.Index)
	}
	return strconv.Itoa(d.Index)
}

// Process is a program using the device.
type Process struct {
	PID     int     `json:"pid"`
	Name    string  `json:"name"`
	Metrics Metrics `json:"metrics"` // MemUsed and Util when known

	// Filled in by the collector from /proc and Kubernetes.
	User      string    `json:"user,omitempty"`
	Command   string    `json:"command,omitempty"`
	Container string    `json:"container,omitempty"`
	Pod       string    `json:"pod,omitempty"`
	Namespace string    `json:"namespace,omitempty"`
	Workload  string    `json:"workload,omitempty"` // "Deployment/chat-api"
	Job       string    `json:"job,omitempty"`      // Slurm job ID
	Started   time.Time `json:"started,omitempty"`
	// App carries metrics the workload published about itself
	// (/run/siltide/app/<pid>.json or $TMPDIR/siltide-app-<pid>.json).
	App map[string]float64 `json:"app,omitempty"`
}

// Sort orders devices by node, vendor, parent, and then index.
func Sort(ds []Device) {
	sort.SliceStable(ds, func(i, j int) bool {
		a, b := ds[i], ds[j]
		if a.Node != b.Node {
			return a.Node < b.Node
		}
		if a.Vendor != b.Vendor {
			return a.Vendor < b.Vendor
		}
		if a.Parent != b.Parent {
			return a.Parent < b.Parent
		}
		return a.Index < b.Index
	})
}
