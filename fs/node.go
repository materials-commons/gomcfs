package fs

import (
	"os/user"
	"strconv"

	"github.com/hanwen/go-fuse/v2/fs"
)

type Node struct {
	fs.Inode
	APIToken string
	MCURL    string
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
