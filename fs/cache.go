package fs

import (
	"github.com/materials-commons/gomcfs/mcapi"
	"sync"
	"time"
)

type Cache struct {
	sync.Map
	Timeout time.Duration
}

type CacheEntry struct {
	Directory *mcapi.MCFile
	Files     []mcapi.MCFile
	Reload    bool
	LastLoad  time.Time
}
