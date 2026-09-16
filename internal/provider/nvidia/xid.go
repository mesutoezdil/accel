package nvidia

import "fmt"

// xid describes an NVIDIA Xid error code. The list covers the codes that
// show up in practice on data-centre parts; unknown codes are reported as
// warnings with their number.
type xid struct {
	severity string
	text     string
}

var xids = map[int]xid{
	13:  {"warning", "graphics engine exception"},
	31:  {"warning", "GPU memory page fault"},
	32:  {"warning", "invalid or corrupted push buffer stream"},
	38:  {"warning", "driver firmware error"},
	43:  {"warning", "GPU stopped processing"},
	45:  {"info", "preemptive cleanup, a process was killed"},
	48:  {"critical", "double-bit ECC error"},
	61:  {"warning", "internal micro-controller breakpoint"},
	62:  {"warning", "internal micro-controller halt"},
	63:  {"critical", "ECC page retirement or row remapping recorded"},
	64:  {"critical", "ECC page retirement or row remapping failed"},
	68:  {"warning", "video processor exception"},
	69:  {"warning", "graphics engine class error"},
	74:  {"critical", "NVLink error"},
	79:  {"critical", "GPU has fallen off the bus"},
	92:  {"info", "high single-bit ECC error rate"},
	94:  {"critical", "contained ECC error"},
	95:  {"critical", "uncontained ECC error"},
	119: {"critical", "GSP RPC timeout"},
	120: {"critical", "GSP error"},
	140: {"warning", "unrecovered ECC error"},
	154: {"critical", "GPU recovery action changed"},
}

// describe returns the severity and message for an Xid.
func describe(code int) (string, string) {
	if x, ok := xids[code]; ok {
		return x.severity, fmt.Sprintf("Xid %d: %s", code, x.text)
	}
	return "warning", fmt.Sprintf("Xid %d", code)
}
