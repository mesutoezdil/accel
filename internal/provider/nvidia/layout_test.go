package nvidia

import (
	"testing"
	"unsafe"
)

// NVML (the NVIDIA Management Library) is a C ABI: every struct siltide hands
// to the driver must match the header's layout byte for byte. The expected
// sizes and offsets below are from `nvml.h` (v12 headers) on 64-bit platforms.
func TestStructLayouts(t *testing.T) {
	check := func(name string, got, want uintptr) {
		t.Helper()
		if got != want {
			t.Errorf("%s = %d, want %d", name, got, want)
		}
	}
	check("sizeof memory", unsafe.Sizeof(memory{}), 24)
	check("sizeof memoryV2", unsafe.Sizeof(memoryV2{}), 40)
	check("sizeof utilization", unsafe.Sizeof(utilization{}), 8)
	check("sizeof pciInfo", unsafe.Sizeof(pciInfo{}), 68)
	check("pciInfo.BusID offset", unsafe.Offsetof(pciInfo{}.BusID), 36)
	check("sizeof processInfo", unsafe.Sizeof(processInfo{}), 24)
	check("processInfo.Mem offset", unsafe.Offsetof(processInfo{}.Mem), 8)
	check("processInfo.CI offset", unsafe.Offsetof(processInfo{}.CI), 20)
	check("sizeof processUtil", unsafe.Sizeof(processUtil{}), 32)
	check("processUtil.Timestamp offset", unsafe.Offsetof(processUtil{}.Timestamp), 8)
	check("sizeof violation", unsafe.Sizeof(violation{}), 16)
	check("sizeof eventData", unsafe.Sizeof(eventData{}), 32)
	check("eventData.GI offset", unsafe.Offsetof(eventData{}.GI), 24)
	check("sizeof fieldValue", unsafe.Sizeof(fieldValue{}), 40)
	check("fieldValue.Value offset", unsafe.Offsetof(fieldValue{}.Value), 32)
}

func TestThrottleBits(t *testing.T) {
	if throttleBits(0x1|0x20) != 0|1<<0|1<<2 { // idle + swThermal
		t.Fatal("idle and thermal bits")
	}
	if throttleBits(0x80) != 1<<1 { // hwBrake maps to power cap
		t.Fatal("power brake")
	}
}

func TestXidCatalog(t *testing.T) {
	if sev, msg := describe(79); sev != "critical" || msg != "Xid 79: GPU has fallen off the bus" {
		t.Fatalf("%s %s", sev, msg)
	}
	if sev, msg := describe(9999); sev != "warning" || msg != "Xid 9999" {
		t.Fatalf("%s %s", sev, msg)
	}
}
