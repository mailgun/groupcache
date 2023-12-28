package groupcache

import "sync"

// workspace holds the "global" state for groupcache.
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

// DefaultWorkspace is the default workspace used by non-workspace-aware APIs.
// If your application does not need to recreate groupcache resources,
// you should use the non-workspace-aware APIs.
// This is likely the most common case.
var DefaultWorkspace = NewWorkspace()

// NewWorkspace creates an explicit workspace for workspace-aware APIs.
// If your application needs to recreate groupcache resources at some
// point, you should use the workspace-aware APIs.
// In order to release current groupcache resources, your application
// would drop all references to the workspace.
func NewWorkspace() *workspace {
	return &workspace{
		groups: make(map[string]*Group),
	}
}
