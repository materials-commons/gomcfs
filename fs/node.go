package fs

import (
	"context"
	"hash/fnv"
	"log"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/materials-commons/gomcfs/mcapi"
)

var (
	_ fs.NodeGetattrer = &Node{}
	_ fs.NodeReaddirer = &Node{}
	_ fs.NodeLookuper  = &Node{}
)

type Node struct {
	fs.Inode
	mcapi       *mcapi.Client
	MCFile      *mcapi.MCFile
	files       []mcapi.MCFile
	filesLoaded bool
}

func RootNode(mcapi *mcapi.Client) *Node {
	var (
		err error
	)
	n := &Node{
		mcapi: mcapi,
	}

	if n.MCFile, err = n.mcapi.GetFileByPath("/"); err != nil {
		log.Panicf("Server not responding: %s, aborting...", err)
	}

	if n.files, err = n.mcapi.ListDirectory("/"); err != nil {
		log.Panicf("Server not responding: %s, aborting...", err)
	}

	n.filesLoaded = true

	return n
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
	//if n.MCFile != nil {
	//	fmt.Printf("MCFile not nil, path = %s\n", n.MCFile.Path)
	//}
	path := n.Path(n.Root())
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	//fmt.Printf("Readdir path: '%s'\n", path)

	var err error
	if !n.filesLoaded {
		if n.files, err = n.mcapi.ListDirectory(path); err != nil {
			//fmt.Printf("   ListDirectory returned error %s for path %s\n", err, path)
			return nil, syscall.EIO
		}
		n.filesLoaded = true
	}

	filesList := make([]fuse.DirEntry, 0, len(n.files))

	for _, fileEntry := range n.files {
		entry := fuse.DirEntry{
			Mode: n.getMode(&fileEntry),
			Name: fileEntry.Name,
			Ino:  n.inodeHash(&fileEntry),
		}

		//fmt.Printf("Entry = %+v\n", entry)
		filesList = append(filesList, entry)
	}

	return fs.NewListDirStream(filesList), fs.OK
}

func (n *Node) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	path := n.Path(n.Root()) + string(filepath.Separator) + name

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	//fmt.Printf("Lookup: '%s': %s\n", path, name)
	var file *mcapi.MCFile
	var err error

	if n.filesLoaded {
		//fmt.Printf("files loaded for path %s, skipping REST call\n", path)
		//fmt.Printf("Looking for %s\n", path)
		for _, fileEntry := range n.files {
			//fmt.Printf("fileEntry.Path = %s\n", fileEntry.Path)
			if pathsMatch(path, fileEntry) {
				file = &fileEntry
				break
			}
		}
	} else {
		file, err = n.mcapi.GetFileByPath(path)
		if err != nil {
			//log.Infof("GetFileByPath returned error %s for path %s", err, path)
			return nil, syscall.ENOENT
		}
	}

	if file == nil {
		//log.Infof("Couldn't find file %s", path)
		return nil, syscall.ENOENT
	}

	//fmt.Printf("%+v\n", file)
	newNode := Node{
		mcapi:  n.mcapi,
		MCFile: file,
	}

	out.Uid = uid
	out.Gid = gid
	if file.IsFile() {
		out.Size = file.Size
	}

	now := time.Now()
	out.SetTimes(&now, &now, &now)

	return n.NewInode(ctx, &newNode, fs.StableAttr{Mode: n.getMode(file), Ino: n.inodeHash(file)}), fs.OK
}

func pathsMatch(path string, fileEntry mcapi.MCFile) bool {
	if fileEntry.IsDir() {
		return path == fileEntry.Path
	}

	//fmt.Printf("%s: fileEntry.Directory.Path = %s\n", fileEntry.Name, fileEntry.Directory.Path)
	if fileEntry.Directory.Path == "/" {
		//fmt.Printf("pathsMatch root: %s %s\n", path, fileEntry.Directory.Path+fileEntry.Name)
		return path == fileEntry.Directory.Path+fileEntry.Name
	}

	//fmt.Printf("pathsMatch sub: %s %s\n", path, fileEntry.Directory.Path+"/"+fileEntry.Name)
	return path == fileEntry.Directory.Path+"/"+fileEntry.Name
}

func (n *Node) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	//path := n.Path(n.Root())
	//if n.MCFile != nil {
	//	fmt.Printf("GetAttr: path = %s name = %s\n", path, n.MCFile.Name)
	//} else {
	//	fmt.Printf("GetAttr MCFile nil: %s\n", path)
	//}
	out.Mode = n.getMode(n.MCFile)
	if n.MCFile == nil {
		out.Size = 0
	} else {
		out.Size = n.MCFile.Size
	}
	out.Ino = n.inodeHash(n.MCFile)
	now := time.Now()
	out.SetTimes(&now, &now, &now)
	out.Uid = uid
	out.Gid = gid
	return fs.OK
}

func (n *Node) getMode(entry *mcapi.MCFile) uint32 {
	if entry == nil {
		return 0755 | uint32(syscall.S_IFDIR)
	}

	if entry.IsDir() {
		return 0755 | uint32(syscall.S_IFDIR)
	}

	//return 0644 | uint32(syscall.S_IFREG)
	return 0666 | uint32(syscall.S_IFREG)
}

func (n *Node) inodeHash(entry *mcapi.MCFile) uint64 {
	if entry == nil {
		//fmt.Printf("inodeHash entry is nil\n")
		return 1
	}

	//fmt.Printf("inodeHash entry.FullPath() = %s\n", entry.FullPath())
	h := fnv.New64a()
	_, _ = h.Write([]byte(entry.FullPath()))
	return h.Sum64()
}
