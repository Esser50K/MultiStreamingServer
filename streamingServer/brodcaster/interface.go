package broadcaster

type BroadcasterIface interface {
	RegisterClient(bc BroadcastClient) error
	Broadcast() error
	Stop() error
}

type BroadcastClient interface {
	GetID() string
	GetStreamID() string
}
