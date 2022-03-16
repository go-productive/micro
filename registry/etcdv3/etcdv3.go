package etcdv3

import (
	"bytes"
	"context"
	"fmt"
	"github.com/go-productive/micro"
	"github.com/go-productive/micro/registry"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	"go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"time"
)

type (
	_DiscoveryRegistry struct {
		client  *clientv3.Client
		options *_Options
	}
)

func New(endpoints []string, opts ...Option) *_DiscoveryRegistry {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: micro.Timeout,
		DialOptions: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		},
	})
	if err != nil {
		panic(err)
	}
	return &_DiscoveryRegistry{
		client:  client,
		options: newOptions(opts),
	}
}

func (d *_DiscoveryRegistry) Register(node *registry.Node) (func(), error) {
	grantRsp, err := d.renewGrant(node)
	if err != nil {
		return nil, err
	}
	deregisterNotifyChan := make(chan struct{})
	go func() {
		ticker := time.NewTicker(d.options.interval)
		defer ticker.Stop()
		for {
			select {
			case <-deregisterNotifyChan:
				return
			case <-ticker.C:
				grantRsp = d.keepalive(node, grantRsp)
			}
		}
	}()
	return func() {
		close(deregisterNotifyChan)
		timeout, cancelFunc := context.WithTimeout(context.TODO(), micro.Timeout)
		defer cancelFunc()
		_, _ = d.client.Delete(timeout, d.toKey(node))
	}, nil
}

func (d *_DiscoveryRegistry) renewGrant(node *registry.Node) (*clientv3.LeaseGrantResponse, error) {
	key, value := d.toKey(node), string(node.Metadata)
	timeout, cancelFunc := context.WithTimeout(context.TODO(), micro.Timeout)
	defer cancelFunc()
	grantRsp, err := d.client.Grant(timeout, int64(d.options.ttl.Seconds()))
	if err != nil {
		return nil, err
	}
	_, err = d.client.Put(timeout, key, value, clientv3.WithLease(grantRsp.ID))
	return grantRsp, err
}

func (d *_DiscoveryRegistry) decodeNode(key, value []byte) (node *registry.Node, err error) {
	split := bytes.Split(key[len(d.options.prefix):], []byte{'/'})
	if len(split) != 3 {
		return nil, fmt.Errorf("illegal key:%v", string(key))
	}
	node = &registry.Node{
		ServiceName: string(split[1]),
		Addr:        string(split[2]),
		Metadata:    value,
	}
	return node, nil
}

func (d *_DiscoveryRegistry) toKey(node *registry.Node) string {
	return d.options.prefix + "/" + node.ServiceName + "/" + node.Addr
}

func (d *_DiscoveryRegistry) keepalive(node *registry.Node, grantRsp *clientv3.LeaseGrantResponse) *clientv3.LeaseGrantResponse {
	timeout, cancelFunc := context.WithTimeout(context.TODO(), micro.Timeout)
	defer cancelFunc()
	for i, t := 0, time.Millisecond*10; i < 5; i, t = i+1, t<<1 {
		_, err := d.client.KeepAliveOnce(timeout, grantRsp.ID)
		if err == nil {
			break
		}
		if err == rpctypes.ErrLeaseNotFound {
			if grantRsp, err = d.renewGrant(node); err == nil {
				break
			}
		}
		d.options.logErrorFunc("keepalive", "err", err, "node", node)
		time.Sleep(t)
	}
	return grantRsp
}

func (d *_DiscoveryRegistry) WatchAndGet() (<-chan *registry.Event, map[string][]*registry.Node, error) {
	watchChan := d.client.Watch(context.TODO(), d.options.prefix, clientv3.WithPrefix())
	eventChan := make(chan *registry.Event, 1)
	go func() {
		defer close(eventChan)
		for watchRsp := range watchChan {
			d.handleWatchRsp(&watchRsp, eventChan)
		}
	}()

	timeout, cancelFunc := context.WithTimeout(context.TODO(), micro.Timeout)
	defer cancelFunc()
	rsp, err := d.client.Get(timeout, d.options.prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, nil, err
	}
	serviceNameMapNodes := make(map[string][]*registry.Node)
	for _, kv := range rsp.Kvs {
		node, err := d.decodeNode(kv.Key, kv.Value)
		if err != nil {
			return nil, nil, err
		}
		serviceNameMapNodes[node.ServiceName] = append(serviceNameMapNodes[node.ServiceName], node)
	}
	return eventChan, serviceNameMapNodes, nil
}

func (d *_DiscoveryRegistry) handleWatchRsp(watchRsp *clientv3.WatchResponse, eventCh chan<- *registry.Event) {
	if watchRsp.Err() != nil {
		d.options.logErrorFunc("handleWatchRsp", "err", watchRsp.Err(), "watchRsp", watchRsp)
		return
	}
	for _, etcdEvent := range watchRsp.Events {
		key, value := etcdEvent.Kv.Key, etcdEvent.Kv.Value
		node, err := d.decodeNode(key, value)
		if err != nil {
			d.options.logErrorFunc("handleWatchRsp", "key", string(key), "value", string(value), "err", err)
			continue
		}
		event := &registry.Event{Node: node}
		switch etcdEvent.Type {
		case clientv3.EventTypePut:
			if etcdEvent.IsCreate() {
				event.Type = registry.NodeEventTypeCreate
			} else {
				event.Type = registry.NodeEventTypeUpdate
			}
		case clientv3.EventTypeDelete:
			event.Type = registry.NodeEventTypeDelete
		}
		eventCh <- event
	}
}
