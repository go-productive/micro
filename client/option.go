package client

import (
	"github.com/go-productive/micro/client/selector"
	"github.com/go-productive/micro/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"runtime"
)

type (
	_Options struct {
		selectorFunc    func(serviceName string) selector.Selector
		dialOptions     []grpc.DialOption
		connSizePerAddr int // latency is slow when high load if only one grpc conn
		onEventFunc     func(event *registry.Event)
		logInfoFunc     func(msg string, keysAndValues ...interface{})
	}
	Option func(*_Options)
)

func newOptions(opts ...Option) *_Options {
	o := &_Options{
		selectorFunc: func(serviceName string) selector.Selector {
			return new(selector.UniversalSelector)
		},
		dialOptions: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		},
		connSizePerAddr: runtime.GOMAXPROCS(0),
		onEventFunc:     func(event *registry.Event) {},
		logInfoFunc: func(msg string, keysAndValues ...interface{}) {
			log.Println(append([]interface{}{"msg", msg}, keysAndValues...)...)
		},
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func WithSelectorFunc(selectorFunc func(serviceName string) selector.Selector) Option {
	return func(o *_Options) {
		o.selectorFunc = selectorFunc
	}
}

func WithDialOptions(dialOptions ...grpc.DialOption) Option {
	return func(o *_Options) {
		o.dialOptions = append(o.dialOptions, dialOptions...)
	}
}

func WithConnSizePerAddr(connSizePerAddr int) Option {
	return func(o *_Options) {
		o.connSizePerAddr = connSizePerAddr
	}
}

func WithOnEventFunc(onEventFunc func(event *registry.Event)) Option {
	return func(o *_Options) {
		o.onEventFunc = onEventFunc
	}
}

func WithLogInfoFunc(logInfoFunc func(msg string, keysAndValues ...interface{})) Option {
	return func(o *_Options) {
		o.logInfoFunc = logInfoFunc
	}
}
