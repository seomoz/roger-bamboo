package configuration

type Bamboo struct {
	// Service host
	Endpoint string

	// enables or disabled the registering of the Marathon event callback
	// the callback is used any time there are application changes within Marathon
	RegisterCallback bool

	// Routing configuration storage
	Zookeeper Zookeeper
}
