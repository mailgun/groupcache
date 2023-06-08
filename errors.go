package groupcache

// ErrNotFound should be returned from an implementation of `GetterFunc` to indicate the
// requested value is not available. When remote HTTP calls are made to retrieve values from
// other groupcache instances, returning this error will indicate to groupcache that the
// value requested is not available, and it should NOT attempt to call `GetterFunc` locally.
type ErrNotFound struct {
	Msg string
}

func (e *ErrNotFound) Error() string {
	return e.Msg
}

func (e *ErrNotFound) Is(target error) bool {
	_, ok := target.(*ErrNotFound)
	return ok
}

// ErrRemoteCall is returned from `group.Get()` when an error that is not a `ErrNotFound`
// is returned during a remote HTTP instance call
type ErrRemoteCall struct {
	Msg string
}

func (e *ErrRemoteCall) Error() string {
	return e.Msg
}

func (e *ErrRemoteCall) Is(target error) bool {
	_, ok := target.(*ErrRemoteCall)
	return ok
}
