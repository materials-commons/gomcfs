package fs

import (
	"context"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"syscall"
)

type MCFSFile struct {
}

func (f *MCFSFile) Read(ctx context.Context, buf []byte, off int64) (res fuse.ReadResult, errno syscall.Errno) {
	return nil, fs.OK
}

func (f *MCFSFile) Write(ctx context.Context, data []byte, off int64) (uint32, syscall.Errno) {
	return 0, fs.OK
}

func (f *MCFSFile) Flush(ctx context.Context) syscall.Errno {
	return fs.OK
}

func (f *MCFSFile) Lseek(ctx context.Context, off uint64, whence uint32) (uint64, syscall.Errno) {
	return 0, fs.OK
}
