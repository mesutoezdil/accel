package mcp

import (
	"bufio"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// ServeStdio reads newline-delimited JSON-RPC from r and writes replies to w,
// until r ends or ctx does. Nothing else may be written to w: a stray line on
// stdout is a protocol error to the client on the other end, which is why
// siltide sends its log elsewhere while this runs.
func (s *Server) ServeStdio(ctx context.Context, r io.Reader, w io.Writer) error {
	in := bufio.NewScanner(r)
	in.Buffer(make([]byte, 0, 64<<10), maxMessage)
	out := json.NewEncoder(w)

	for in.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := strings.TrimSpace(in.Text())
		if line == "" {
			continue
		}
		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			// No id to answer with, so this is the one case that gets a bare
			// error object rather than a reply to something.
			if err := out.Encode(Response{JSONRPC: "2.0", Error: &RPCError{Code: -32700, Message: "parse error"}}); err != nil {
				return err
			}
			continue
		}
		resp := s.Handle(req)
		if resp == nil {
			continue // a notification is answered by saying nothing
		}
		if err := out.Encode(resp); err != nil {
			return err
		}
	}
	return in.Err()
}

// maxMessage is the largest request accepted on either transport. A snapshot
// of a large fleet is tens of kilobytes; a megabyte is past anything a client
// has reason to send and short of anything worth buffering.
const maxMessage = 1 << 20

// Handler serves the same server over HTTP, for clients that speak the
// streamable transport rather than stdio. It answers POSTs at the root.
//
// The checks are what a loopback endpoint needs and no more: the bearer token
// --listen already uses, a Host header that resolves to a loopback address so
// a browser on the same machine cannot be pointed at it by a page, and a
// bounded body.
func (s *Server) Handler(token string) http.Handler {
	want := strings.TrimPrefix(token, "sha256:")
	if token != "" && !strings.HasPrefix(token, "sha256:") {
		want = digest(token)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			http.Error(w, "the Model Context Protocol endpoint takes POST", http.StatusMethodNotAllowed)
			return
		}
		if !loopbackHost(r.Host) {
			http.Error(w, "this endpoint is loopback only", http.StatusForbidden)
			return
		}
		if want != "" {
			got := digest(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
			if subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
				w.Header().Set("WWW-Authenticate", "Bearer")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}

		var req Request
		if err := json.NewDecoder(io.LimitReader(r.Body, maxMessage)).Decode(&req); err != nil {
			writeJSON(w, Response{JSONRPC: "2.0", Error: &RPCError{Code: -32700, Message: "parse error"}})
			return
		}
		resp := s.Handle(req)
		if resp == nil {
			w.WriteHeader(http.StatusAccepted) // a notification: nothing to say back
			return
		}
		writeJSON(w, resp)
	})
}

// Serve runs the HTTP transport on addr until ctx ends. addr must be
// loopback: this is a local endpoint for a client on the same machine.
func (s *Server) Serve(ctx context.Context, addr, token string) error {
	if !loopbackAddr(addr) {
		return errors.New("--mcp-http takes a loopback address, such as 127.0.0.1:8765")
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(token),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func digest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// loopbackAddr reports whether a listen address stays on this machine.
func loopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	return isLoopbackName(host)
}

// loopbackHost reports whether a Host header names this machine. A request
// arriving with any other name is one a page sent here on someone's behalf.
func loopbackHost(host string) bool {
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return isLoopbackName(host)
}

func isLoopbackName(host string) bool {
	host = strings.Trim(host, "[]")
	switch strings.ToLower(host) {
	case "localhost", "":
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
