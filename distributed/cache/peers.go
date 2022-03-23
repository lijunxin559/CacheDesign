package cache

type PeerPicker interface {
	//select Peer based on key
	PickPeer(key string) (peer PeerGetter, ok bool)
}

type PeerGetter interface {
	//get value based on peer and key
	Get(group string, key string) ([]byte, error)
}
