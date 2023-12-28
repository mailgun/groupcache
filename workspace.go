package groupcache

import "sync"

type workspace struct {
	httpPoolMade bool
	portPicker   func(groupName string) PeerPicker

	mu     sync.RWMutex
	groups map[string]*Group

	initPeerServerOnce sync.Once
	initPeerServer     func()

	// newGroupHook, if non-nil, is called right after a new group is created.
	newGroupHook func(*Group)
}

var DefaultWorkspace = NewWorkspace()

func NewWorkspace() *workspace {
	return &workspace{
		groups: make(map[string]*Group),
	}
}
