package kube

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// The kubelet's pod-resources API tells which pod holds which device even
// when nothing is running on it: a pod that requested a device and sleeps
// still owns it. It is gRPC over a unix socket; siltide speaks just enough
// HTTP/2 and protobuf to call List, so no gRPC dependency is pulled in.

// Allocation is one device handed to a container.
type Allocation struct {
	Namespace string
	Pod       string
	Container string
	Resource  string // "nvidia.com/gpu"
	DeviceID  string // what the device plugin registered: UUID, index, bus
}

// PodResources reads the kubelet socket.
type PodResources struct {
	mu     sync.Mutex
	socket string
	client *http.Client
	at     time.Time
	ttl    time.Duration
	cached []Allocation
	err    error
}

// NewPodResources returns a client; nil when the socket does not exist.
func NewPodResources() *PodResources {
	for _, s := range []string{os.Getenv("SILTIDE_POD_RESOURCES_SOCKET"), "/var/lib/kubelet/pod-resources/kubelet.sock"} {
		if s == "" {
			continue
		}
		if _, err := os.Stat(s); err == nil {
			return newPodResources(s)
		}
	}
	return nil
}

func newPodResources(socket string) *PodResources {
	tr := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", socket)
		},
		Protocols: new(http.Protocols),
	}
	tr.Protocols.SetHTTP1(false)
	tr.Protocols.SetUnencryptedHTTP2(true)
	return &PodResources{socket: socket, client: &http.Client{Transport: tr, Timeout: 5 * time.Second}, ttl: 15 * time.Second}
}

// Allocations lists every device allocation on the node.
func (p *PodResources) Allocations(ctx context.Context) ([]Allocation, error) {
	if p == nil {
		return nil, nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if time.Since(p.at) < p.ttl {
		return p.cached, p.err
	}
	p.at = time.Now()
	p.cached, p.err = p.list(ctx)
	return p.cached, p.err
}

func (p *PodResources) list(ctx context.Context) ([]Allocation, error) {
	// gRPC request: 1-byte compression flag, 4-byte length, then an empty
	// ListPodResourcesRequest message.
	body := []byte{0, 0, 0, 0, 0}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://kubelet/v1.PodResourcesLister/List", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/grpc")
	req.Header.Set("TE", "trailers")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pod-resources: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if st := resp.Trailer.Get("Grpc-Status"); st != "" && st != "0" {
		return nil, fmt.Errorf("pod-resources: grpc status %s %s", st, resp.Trailer.Get("Grpc-Message"))
	}
	if len(raw) < 5 {
		return nil, errors.New("pod-resources: short reply")
	}
	n := binary.BigEndian.Uint32(raw[1:5])
	if int(n) > len(raw)-5 {
		return nil, errors.New("pod-resources: truncated reply")
	}
	return ParseListResponse(raw[5 : 5+n])
}

// ParseListResponse decodes a ListPodResourcesResponse:
//
//	message ListPodResourcesResponse { repeated PodResources pod_resources = 1; }
//	message PodResources { string name = 1; string namespace = 2; repeated ContainerResources containers = 3; }
//	message ContainerResources { string name = 1; repeated ContainerDevices devices = 2; }
//	message ContainerDevices { string resource_name = 1; repeated string device_ids = 2; }
func ParseListResponse(b []byte) ([]Allocation, error) {
	var out []Allocation
	err := walk(b, func(field int, v []byte) error {
		if field != 1 {
			return nil
		}
		var pod, ns string
		var containers [][]byte
		if err := walk(v, func(f int, v []byte) error {
			switch f {
			case 1:
				pod = string(v)
			case 2:
				ns = string(v)
			case 3:
				containers = append(containers, v)
			}
			return nil
		}); err != nil {
			return err
		}
		for _, c := range containers {
			var cname string
			var devs [][]byte
			if err := walk(c, func(f int, v []byte) error {
				switch f {
				case 1:
					cname = string(v)
				case 2:
					devs = append(devs, v)
				}
				return nil
			}); err != nil {
				return err
			}
			for _, d := range devs {
				var res string
				var ids []string
				if err := walk(d, func(f int, v []byte) error {
					switch f {
					case 1:
						res = string(v)
					case 2:
						ids = append(ids, string(v))
					}
					return nil
				}); err != nil {
					return err
				}
				for _, id := range ids {
					out = append(out, Allocation{Namespace: ns, Pod: pod, Container: cname, Resource: res, DeviceID: id})
				}
			}
		}
		return nil
	})
	return out, err
}

// walk visits every length-delimited field of a protobuf message; other
// wire types are skipped.
func walk(b []byte, visit func(field int, v []byte) error) error {
	for len(b) > 0 {
		key, n := binary.Uvarint(b)
		if n <= 0 {
			return errors.New("protobuf: bad key")
		}
		b = b[n:]
		field, wire := int(key>>3), key&7
		switch wire {
		case 0: // varint
			_, n := binary.Uvarint(b)
			if n <= 0 {
				return errors.New("protobuf: bad varint")
			}
			b = b[n:]
		case 1:
			if len(b) < 8 {
				return errors.New("protobuf: short fixed64")
			}
			b = b[8:]
		case 2:
			l, n := binary.Uvarint(b)
			if n <= 0 || uint64(len(b)-n) < l {
				return errors.New("protobuf: bad length")
			}
			if err := visit(field, b[n:n+int(l)]); err != nil {
				return err
			}
			b = b[n+int(l):]
		case 5:
			if len(b) < 4 {
				return errors.New("protobuf: short fixed32")
			}
			b = b[4:]
		default:
			return fmt.Errorf("protobuf: wire type %d", wire)
		}
	}
	return nil
}

// Encode helpers for tests: build a message from fields.
func field(num int, v []byte) []byte {
	out := binary.AppendUvarint(nil, uint64(num<<3|2))
	out = binary.AppendUvarint(out, uint64(len(v)))
	return append(out, v...)
}
