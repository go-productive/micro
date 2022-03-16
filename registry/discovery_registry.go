package registry

const (
	NodeEventTypeCreate nodeEventType = "create"
	NodeEventTypeUpdate nodeEventType = "update"
	NodeEventTypeDelete nodeEventType = "delete"
)

type (
	Node struct {
		ServiceName string
		Addr        string
		Metadata    []byte
	}
	nodeEventType string
	Event         struct {
		Type nodeEventType
		Node *Node
	}
	Discovery interface {
		WatchAndGet() (<-chan *Event, map[string][]*Node, error)
	}
	Registry interface {
		Register(node *Node) (deregisterFunc func(), err error)
	}
	DiscoveryRegistry interface {
		Discovery
		Registry
	}
)
