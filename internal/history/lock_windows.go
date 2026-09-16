//go:build windows

package history

import "os"

func acquire(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644) // ponytail: no advisory lock on Windows
}

func release(f *os.File) { _ = f.Close() }
