package fs

import (
	"context"
	"hash/fnv"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/materials-commons/gomcfs/mcapi"
)

type Node struct {
	fs.Inode
	mcapi  mcapi.Client
	MCFile *mcapi.MCFile
}

// Set file owners to the current user,
// otherwise in OSX, we will fail to start.
var uid, gid uint32

func init() {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	uid32, _ := strconv.ParseUint(u.Uid, 10, 32)
	gid32, _ := strconv.ParseUint(u.Gid, 10, 32)
	uid = uint32(uid32)
	gid = uint32(gid32)
}

func (n *Node) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	path := n.Path(n.Root())
	files, err := n.mcapi.ListDirectory(path)
	if err != nil {
		return nil, syscall.EIO
	}

	filesList := make([]fuse.DirEntry, 0, len(files))

	for _, fileEntry := range files {
		entry := fuse.DirEntry{
			Mode: n.getMode(fileEntry),
			Name: fileEntry.Name,
			Ino:  n.inodeHash(fileEntry),
		}

		filesList = append(filesList, entry)
	}

	return fs.NewListDirStream(filesList), fs.OK
}

func (n *Node) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	path := n.Path(n.Root()) + string(filepath.Separator) + name

	file, err := n.mcapi.GetFileByPath(path)
	if err != nil {
		return nil, syscall.ENOENT
	}

	newNode := Node{
		mcapi:  n.mcapi,
		MCFile: file,
	}

	return n.NewInode(ctx, &newNode, fs.StableAttr{Mode: n.getMode(*file), Ino: n.inodeHash(*file)}), fs.OK
}

func (n *Node) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = n.getMode(*n.MCFile)
	out.Size = n.MCFile.Size
	out.Ino = n.inodeHash(*n.MCFile)
	now := time.Now()
	out.SetTimes(&now, &now, &now)
	out.Uid = uid
	out.Gid = gid
	return fs.OK
}

func (n *Node) getMode(entry mcapi.MCFile) uint32 {
	if entry.IsDir() {
		return 0755 | uint32(syscall.S_IFDIR)
	}

	return 0644 | uint32(syscall.S_IFREG)
}

func (n *Node) inodeHash(entry mcapi.MCFile) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(entry.FullPath()))
	return h.Sum64()
}
