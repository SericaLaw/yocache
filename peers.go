package yocache

// PeerPicker is the interface that must be implemented to locate
// the peer that owns a specific key.
type PeerPicker interface {
	// PickPeer returns a PeerGetter which can fetch keys from the peer.
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter is the interface that must be implemented by a peer to get
// value owned by it.
type PeerGetter interface {
	Get(group string, key string) (value []byte, err error)
}
