package selector

import (
	"context"
	"github.com/go-productive/micro/registry"
	"math/rand"
	"sync"
	"sync/atomic"
)

type (
	Selector interface {
		OnInit([]*registry.Node)
		OnEvent(event *registry.Event)
		Select(ctx context.Context) *registry.Node
		Nodes() []*registry.Node
	}
	UniversalSelector struct {
		rwMutex     sync.RWMutex
		nodes       []*registry.Node
		consistHash _ConsistHash

		sequence uint64
	}
)

func (u *UniversalSelector) OnInit(nodes []*registry.Node) {
	u.rwMutex.Lock()
	defer u.rwMutex.Unlock()
	for _, node := range nodes {
		u.addNode(node)
	}
	u.resetConsistHash()
}

func (u *UniversalSelector) addNode(addNode *registry.Node) {
	cp := make([]*registry.Node, 0, len(u.nodes)+1)
	for _, node := range u.nodes {
		if node.Addr != addNode.Addr {
			cp = append(cp, node)
		}
	}
	cp = append(cp, addNode)
	u.nodes = cp
}

func (u *UniversalSelector) remNode(remNode *registry.Node) {
	cp := make([]*registry.Node, 0, len(u.nodes)-1)
	for _, node := range u.nodes {
		if node.Addr != remNode.Addr {
			cp = append(cp, node)
		}
	}
	u.nodes = cp
}

func (u *UniversalSelector) OnEvent(event *registry.Event) {
	u.rwMutex.Lock()
	defer u.rwMutex.Unlock()
	switch event.Type {
	case registry.NodeEventTypeCreate, registry.NodeEventTypeUpdate:
		u.addNode(event.Node)
	case registry.NodeEventTypeDelete:
		u.remNode(event.Node)
	}
	u.resetConsistHash()
}

func (u *UniversalSelector) Select(ctx context.Context) *registry.Node {
	u.rwMutex.RLock()
	defer u.rwMutex.RUnlock()
	if len(u.nodes) <= 0 {
		return nil
	}
	if ctx.Value(specifyAddr{}) != nil {
		for _, node := range u.nodes {
			if node.Addr == ctx.Value(specifyAddr{}).(string) {
				return node
			}
		}
		return nil
	}
	hashKey := ctx.Value(consistHash{})
	switch {
	case hashKey != nil:
		return u.consistHash.get(hashKey.(string))
	case ctx.Value(roundRobin{}) != nil:
		return u.nodes[atomic.AddUint64(&u.sequence, 1)%uint64(len(u.nodes))]
	default:
		return u.nodes[rand.Intn(len(u.nodes))]
	}
}

func (u *UniversalSelector) Nodes() []*registry.Node {
	u.rwMutex.RLock()
	defer u.rwMutex.RUnlock()
	return u.nodes
}
