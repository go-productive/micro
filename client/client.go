package client

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-productive/micro"
	"github.com/go-productive/micro/client/selector"
	"github.com/go-productive/micro/registry"
	"google.golang.org/grpc"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"
)

var (
	ErrNonstandardGRPCMethod = errors.New("nonstandard grpc method")
)

type (
	Client struct {
		clientConn *grpc.ClientConn
		options    *_Options
		discovery  registry.Discovery

		selectorRWMutex        sync.RWMutex
		serviceNameMapSelector map[string]selector.Selector

		connSetRWMutex sync.RWMutex
		addrMapConnSet map[string]*_ConnSet
	}
	_ConnSet struct {
		sequence    uint64
		connections []*grpc.ClientConn
	}
)

func New(discovery registry.Discovery, opts ...Option) *Client {
	c := &Client{
		options:                newOptions(opts...),
		discovery:              discovery,
		serviceNameMapSelector: make(map[string]selector.Selector),
		addrMapConnSet:         make(map[string]*_ConnSet),
	}
	c.initClientConn()
	c.initServicesThenWatch()
	return c
}

func (c *Client) ClientConn() *grpc.ClientConn {
	return c.clientConn
}

func (c *Client) Selector(serviceName string) selector.Selector {
	return c.getOrCreateSelector(serviceName)
}

func (c *Client) initClientConn() {
	doptsField, ok := reflect.TypeOf(grpc.ClientConn{}).FieldByName("dopts")
	if !ok {
		panic("no dopts, compatible grpc version")
	}
	unaryIntField, ok := doptsField.Type.FieldByName("unaryInt")
	if !ok {
		panic("no dopts.unaryInt, compatible grpc version")
	}
	streamIntField, ok := doptsField.Type.FieldByName("streamInt")
	if !ok {
		panic("no dopts.streamInt, compatible grpc version")
	}
	c.clientConn = new(grpc.ClientConn)
	dopts := reflect.ValueOf(c.clientConn).Elem().FieldByName("dopts")
	*(*grpc.UnaryClientInterceptor)(unsafe.Pointer(dopts.UnsafeAddr() + unaryIntField.Offset)) = c.unaryInterceptor
	*(*grpc.StreamClientInterceptor)(unsafe.Pointer(dopts.UnsafeAddr() + streamIntField.Offset)) = c.streamInterceptor
}

func (c *Client) unaryInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	if ctx == nil {
		ctx = context.TODO()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancelFunc func()
		ctx, cancelFunc = context.WithTimeout(ctx, micro.Timeout)
		defer cancelFunc()
	}
	clientConn, err := c.selectClientConn(ctx, method)
	if err != nil {
		return err
	}
	return clientConn.Invoke(ctx, method, req, reply, opts...)
}

func (c *Client) streamInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (stream grpc.ClientStream, e error) {
	if ctx == nil {
		ctx = context.TODO()
	}
	clientConn, err := c.selectClientConn(ctx, method)
	if err != nil {
		return nil, err
	}
	return clientConn.NewStream(ctx, desc, method, opts...)
}

func (c *Client) selectClientConn(ctx context.Context, method string) (*grpc.ClientConn, error) {
	split := strings.Split(method, "/")
	if len(split) != 3 {
		return nil, ErrNonstandardGRPCMethod
	}
	node := c.getOrCreateSelector(split[1]).Select(ctx)
	if node == nil {
		return nil, fmt.Errorf("service:%v not found", split[1])
	}
	return c.getOrCreateClientConn(node.Addr)
}

func (c *Client) getOrCreateClientConn(addr string) (*grpc.ClientConn, error) {
	c.connSetRWMutex.RLock()
	connSet, ok := c.addrMapConnSet[addr]
	c.connSetRWMutex.RUnlock()
	if ok {
		return connSet.get(), nil
	}
	c.connSetRWMutex.Lock()
	defer c.connSetRWMutex.Unlock()
	connSet, ok = c.addrMapConnSet[addr]
	if ok { //double check
		return connSet.get(), nil
	}
	connSet, err := c.newConnSet(addr)
	if err != nil {
		return nil, err
	}
	c.addrMapConnSet[addr] = connSet
	return connSet.get(), nil
}

func (c *Client) getOrCreateSelector(serviceName string) selector.Selector {
	c.selectorRWMutex.RLock()
	selector, ok := c.serviceNameMapSelector[serviceName]
	c.selectorRWMutex.RUnlock()
	if ok {
		return selector
	}
	c.selectorRWMutex.Lock()
	defer c.selectorRWMutex.Unlock()
	selector, ok = c.serviceNameMapSelector[serviceName]
	if ok { //double check
		return selector
	}
	selector = c.options.selectorFunc(serviceName)
	c.serviceNameMapSelector[serviceName] = selector
	return selector
}

func (c *Client) initServicesThenWatch() {
	eventCh, serviceNameMapNodes, err := c.discovery.WatchAndGet()
	if err != nil {
		panic(err)
	}
	for serviceName, nodes := range serviceNameMapNodes {
		c.getOrCreateSelector(serviceName).OnInit(nodes)
		c.options.logInfoFunc("initServices", "serviceName", serviceName, "nodes", nodes)
	}
	go func() {
		for event := range eventCh {
			c.options.logInfoFunc("handleEvent", "event", event)
			c.handleEvent(event)
		}
	}()
}

func (c *Client) handleEvent(event *registry.Event) {
	node := event.Node
	c.getOrCreateSelector(node.ServiceName).OnEvent(event)
	if event.Type == registry.NodeEventTypeDelete {
		c.connSetRWMutex.Lock()
		defer c.connSetRWMutex.Unlock()
		if connSet, ok := c.addrMapConnSet[node.Addr]; ok {
			connSet.close()
			delete(c.addrMapConnSet, node.Addr)
		}
	}
	c.options.onEventFunc(event)
}

func (c *Client) newConnSet(addr string) (connSet *_ConnSet, err error) {
	defer func() {
		if err != nil {
			connSet.close()
		}
	}()
	connSet = &_ConnSet{
		connections: make([]*grpc.ClientConn, 0, c.options.connSizePerAddr),
	}
	timeout, cancelFunc := context.WithTimeout(context.TODO(), micro.Timeout)
	defer cancelFunc()
	for i := 0; i < cap(connSet.connections); i++ {
		conn, err := grpc.DialContext(timeout, addr, c.options.dialOptions...)
		if err != nil {
			return connSet, err
		}
		connSet.connections = append(connSet.connections, conn)
	}
	return connSet, nil
}

func (c *_ConnSet) get() *grpc.ClientConn {
	return c.connections[atomic.AddUint64(&c.sequence, 1)%uint64(len(c.connections))]
}

func (c *_ConnSet) close() {
	for _, conn := range c.connections {
		_ = conn.Close()
	}
}
