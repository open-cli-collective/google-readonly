package zip

import (
	"io"
	"os"
)

// FileSystem abstracts filesystem operations for testing
type FileSystem interface {
	MkdirAll(path string, perm os.FileMode) error
	OpenFile(name string, flag int, perm os.FileMode) (io.WriteCloser, error)
	Remove(name string) error
}

// osFS is the real filesystem implementation
type osFS struct{}

func (osFS) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (osFS) OpenFile(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	return os.OpenFile(name, flag, perm)
}

func (osFS) Remove(name string) error {
	return os.Remove(name)
}

// fs is the package-level filesystem, swappable in tests
var fs FileSystem = osFS{}
