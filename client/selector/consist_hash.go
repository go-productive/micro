package selector

import (
	"encoding/binary"
	"github.com/go-productive/micro/registry"
	"github.com/spaolacci/murmur3"
	"math/rand"
	"sort"
)

type (
	_VirtualNode struct {
		hash uint32
		node *registry.Node
	}
	_ConsistHash []*_VirtualNode
)

func (c _ConsistHash) Len() int {
	return len(c)
}

func (c _ConsistHash) Less(i, j int) bool {
	return c[i].hash < c[j].hash
}

func (c _ConsistHash) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (u *UniversalSelector) resetConsistHash() {
	const virtualNodeCount = 128
	u.consistHash = make(_ConsistHash, 0, len(u.nodes)*virtualNodeCount)
	for _, node := range u.nodes {
		bs := make([]byte, 8+len(node.Addr))
		copy(bs[8:], node.Addr)
		random := rand.New(rand.NewSource(0))
		for i := 0; i < virtualNodeCount; i++ {
			binary.BigEndian.PutUint64(bs[:8], random.Uint64())
			u.consistHash = append(u.consistHash, &_VirtualNode{
				hash: murmur3.Sum32(bs),
				node: node,
			})
		}
	}
	sort.Sort(u.consistHash)
}

func (c _ConsistHash) get(hashKey string) *registry.Node {
	if len(c) <= 0 {
		return nil
	}
	sum32 := murmur3.Sum32([]byte(hashKey))
	search := sort.Search(len(c), func(i int) bool {
		return c[i].hash > sum32
	})
	if search >= len(c) {
		search = 0
	}
	return c[search].node
}
