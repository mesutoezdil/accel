//go:build darwin && arm64

package apple

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
)

// Power and energy come from IOReport's "Energy Model" group, temperature
// from the GPU keys of the SMC (System Management Controller), both bound at
// runtime with purego so the binary stays cgo-free and needs no root.
// Everything here is best effort: a missing symbol or key leaves the metric
// absent.

var (
	deepOnce sync.Once
	deepErr  error

	cfRelease            func(uintptr)
	cfStringCreate       func(alloc uintptr, s string, enc uint32) uintptr
	cfStringGetCString   func(s uintptr, buf *byte, n int64, enc uint32) bool
	cfDictGetValue       func(dict, key uintptr) uintptr
	cfArrayGetCount      func(a uintptr) int64
	cfArrayGetValueAt    func(a uintptr, i int64) uintptr
	ioReportCopyChannels func(group, sub uintptr, a, b, c uint64) uintptr
	ioReportCreateSub    func(alloc, channels uintptr, subbed *uintptr, channelID uint64, opts uintptr) uintptr
	ioReportCreateSample func(sub, channels, opts uintptr) uintptr
	ioReportDelta        func(prev, cur, opts uintptr) uintptr
	ioReportChannelName  func(ch uintptr) uintptr
	ioReportUnitLabel    func(ch uintptr) uintptr
	ioReportSimpleInt    func(ch uintptr, idx int32) int64

	ioMainPort              uint32
	ioServiceMatching       func(name string) uintptr
	ioServiceGetMatching    func(port uint32, matching uintptr) uint32
	ioServiceOpen           func(service, task uint32, typ uint32, conn *uint32) int32
	ioServiceClose          func(conn uint32) int32
	ioConnectCallStruct     func(conn uint32, sel uint32, in unsafe.Pointer, inSize uint64, out unsafe.Pointer, outSize *uint64) int32
	machTaskSelf            func() uint32
	energySub, energySubbed uintptr
	energyPrev              uintptr
	energyAt                time.Time
	smcConn                 uint32
	smcKeys                 []uint32
)

const cfStringUTF8 = 0x08000100

func loadDeep() error {
	deepOnce.Do(func() {
		cf, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			deepErr = err
			return
		}
		iokit, err := purego.Dlopen("/System/Library/Frameworks/IOKit.framework/IOKit", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			deepErr = err
			return
		}
		sys, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			deepErr = err
			return
		}
		purego.RegisterLibFunc(&cfRelease, cf, "CFRelease")
		purego.RegisterLibFunc(&cfStringCreate, cf, "CFStringCreateWithCString")
		purego.RegisterLibFunc(&cfStringGetCString, cf, "CFStringGetCString")
		purego.RegisterLibFunc(&cfDictGetValue, cf, "CFDictionaryGetValue")
		purego.RegisterLibFunc(&cfArrayGetCount, cf, "CFArrayGetCount")
		purego.RegisterLibFunc(&cfArrayGetValueAt, cf, "CFArrayGetValueAtIndex")
		purego.RegisterLibFunc(&ioServiceMatching, iokit, "IOServiceMatching")
		purego.RegisterLibFunc(&ioServiceGetMatching, iokit, "IOServiceGetMatchingService")
		purego.RegisterLibFunc(&ioServiceOpen, iokit, "IOServiceOpen")
		purego.RegisterLibFunc(&ioServiceClose, iokit, "IOServiceClose")
		purego.RegisterLibFunc(&ioConnectCallStruct, iokit, "IOConnectCallStructMethod")
		purego.RegisterLibFunc(&machTaskSelf, sys, "mach_task_self")
		if rep, err := purego.Dlopen("/usr/lib/libIOReport.dylib", purego.RTLD_NOW|purego.RTLD_GLOBAL); err == nil {
			purego.RegisterLibFunc(&ioReportCopyChannels, rep, "IOReportCopyChannelsInGroup")
			purego.RegisterLibFunc(&ioReportCreateSub, rep, "IOReportCreateSubscription")
			purego.RegisterLibFunc(&ioReportCreateSample, rep, "IOReportCreateSamples")
			purego.RegisterLibFunc(&ioReportDelta, rep, "IOReportCreateSamplesDelta")
			purego.RegisterLibFunc(&ioReportChannelName, rep, "IOReportChannelGetChannelName")
			purego.RegisterLibFunc(&ioReportUnitLabel, rep, "IOReportChannelGetUnitLabel")
			purego.RegisterLibFunc(&ioReportSimpleInt, rep, "IOReportSimpleGetIntegerValue")
			group := cfStringCreate(0, "Energy Model", cfStringUTF8)
			channels := ioReportCopyChannels(group, 0, 0, 0, 0)
			cfRelease(group)
			if channels != 0 {
				energySub = ioReportCreateSub(0, channels, &energySubbed, 0, 0)
				if energySub != 0 {
					energyPrev = ioReportCreateSample(energySub, energySubbed, 0)
					energyAt = time.Now()
				}
			}
		}
		openSMC()
	})
	return deepErr
}

// power returns GPU watts and joules since the previous call.
func power() (watts, joules float64, ok bool) {
	if energySub == 0 || energyPrev == 0 {
		return 0, 0, false
	}
	cur := ioReportCreateSample(energySub, energySubbed, 0)
	if cur == 0 {
		return 0, 0, false
	}
	delta := ioReportDelta(energyPrev, cur, 0)
	now := time.Now()
	dt := now.Sub(energyAt).Seconds()
	cfRelease(energyPrev)
	energyPrev, energyAt = cur, now
	if delta == 0 {
		return 0, 0, false
	}
	defer cfRelease(delta)
	key := cfStringCreate(0, "IOReportChannels", cfStringUTF8)
	arr := cfDictGetValue(delta, key)
	cfRelease(key)
	if arr == 0 {
		return 0, 0, false
	}
	n := cfArrayGetCount(arr)
	for i := int64(0); i < n; i++ {
		ch := cfArrayGetValueAt(arr, i)
		if cfString(ioReportChannelName(ch)) != "GPU Energy" {
			continue
		}
		v := float64(ioReportSimpleInt(ch, 0))
		switch cfString(ioReportUnitLabel(ch)) {
		case "mJ":
			v /= 1e3
		case "uJ", "µJ":
			v /= 1e6
		case "nJ":
			v /= 1e9
		}
		if dt <= 0 {
			return 0, v, true
		}
		return v / dt, v, true
	}
	return 0, 0, false
}

func cfString(s uintptr) string {
	if s == 0 {
		return ""
	}
	buf := make([]byte, 256)
	if !cfStringGetCString(s, &buf[0], int64(len(buf)), cfStringUTF8) {
		return ""
	}
	if i := strings.IndexByte(string(buf), 0); i >= 0 {
		return string(buf[:i])
	}
	return string(buf)
}

// smcKeyData mirrors the 80-byte SMCKeyData_t the AppleSMC user client
// takes for kSMCHandleYPCEvent (selector 2).
type smcKeyData struct {
	key      uint32
	vers     [6]byte
	_        [2]byte
	pLimit   [16]byte
	dataSize uint32
	dataType uint32
	dataAttr uint8
	_        [3]byte
	result   uint8
	status   uint8
	data8    uint8
	_        [1]byte
	data32   uint32
	bytes    [32]byte
}

const (
	smcReadKey      = 5
	smcGetKeyFromIx = 8
	smcGetKeyInfo   = 9
)

func fourcc(s string) uint32 {
	b := []byte(s)
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

func openSMC() {
	svc := ioServiceGetMatching(ioMainPort, ioServiceMatching("AppleSMC"))
	if svc == 0 {
		return
	}
	if ioServiceOpen(svc, machTaskSelf(), 0, &smcConn) != 0 {
		return
	}
	// count keys, then find the GPU temperature keys ("Tg" prefix, float)
	var in, out smcKeyData
	in.key, in.data8 = fourcc("#KEY"), smcGetKeyInfo
	if !smcCall(&in, &out) {
		return
	}
	in = smcKeyData{key: fourcc("#KEY"), data8: smcReadKey, dataSize: out.dataSize, dataType: out.dataType}
	if !smcCall(&in, &out) {
		return
	}
	count := uint32(out.bytes[0])<<24 | uint32(out.bytes[1])<<16 | uint32(out.bytes[2])<<8 | uint32(out.bytes[3])
	for i := uint32(0); i < count && i < 4096; i++ {
		in = smcKeyData{data8: smcGetKeyFromIx, data32: i}
		if !smcCall(&in, &out) {
			continue
		}
		k := out.key
		name := string([]byte{byte(k >> 24), byte(k >> 16), byte(k >> 8), byte(k)})
		if !strings.HasPrefix(name, "Tg") {
			continue
		}
		in = smcKeyData{key: k, data8: smcGetKeyInfo}
		if smcCall(&in, &out) && out.dataType == fourcc("flt ") {
			smcKeys = append(smcKeys, k)
		}
	}
}

func smcCall(in, out *smcKeyData) bool {
	size := uint64(unsafe.Sizeof(*out))
	return ioConnectCallStruct(smcConn, 2, unsafe.Pointer(in), uint64(unsafe.Sizeof(*in)), unsafe.Pointer(out), &size) == 0 && out.result == 0
}

// temperature averages the GPU die sensors.
func temperature() (float64, bool) {
	if smcConn == 0 || len(smcKeys) == 0 {
		return 0, false
	}
	sum, n := 0.0, 0
	for _, k := range smcKeys {
		var in, out smcKeyData
		in.key, in.data8 = k, smcGetKeyInfo
		if !smcCall(&in, &out) {
			continue
		}
		size := out.dataSize
		in = smcKeyData{key: k, data8: smcReadKey, dataSize: size, dataType: out.dataType}
		if !smcCall(&in, &out) || size < 4 {
			continue
		}
		bits := uint32(out.bytes[0]) | uint32(out.bytes[1])<<8 | uint32(out.bytes[2])<<16 | uint32(out.bytes[3])<<24
		v := float64(*(*float32)(unsafe.Pointer(&bits)))
		if v > 0 && v < 130 {
			sum += v
			n++
		}
	}
	if n == 0 {
		return 0, false
	}
	return sum / float64(n), true
}

var (
	reCreator = regexp.MustCompile(`"IOUserClientCreator" = "pid (\d+), ([^"]*)"`)
	reGPUTime = regexp.MustCompile(`"accumulatedGPUTime"\s*=\s*(\d+)`)
)

// gpuTime lists per-process accumulated GPU time (ns) from the AGX user
// clients in the IORegistry.
func gpuTime() map[int]procTime {
	out, err := exec.Command("ioreg", "-r", "-c", "AGXDeviceUserClient").Output()
	if err != nil {
		return nil
	}
	res := map[int]procTime{}
	for _, blk := range strings.Split(string(out), "+-o ")[1:] {
		c := reCreator.FindStringSubmatch(blk)
		if c == nil {
			continue
		}
		pid, _ := strconv.Atoi(c[1])
		t := res[pid]
		t.name = c[2]
		for _, g := range reGPUTime.FindAllStringSubmatch(blk, -1) {
			ns, _ := strconv.ParseFloat(g[1], 64)
			t.ns += ns
		}
		res[pid] = t
	}
	return res
}

type procTime struct {
	name string
	ns   float64
}
