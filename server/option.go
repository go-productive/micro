package server

import (
	"google.golang.org/grpc"
	"log"
	"time"
)

type (
	_Options struct {
		serverOptions         []grpc.ServerOption
		metadata              []byte
		shutdownSleepDuration time.Duration
		logInfoFunc           func(msg string, keysAndValues ...interface{})
	}
	Option func(*_Options)
)

func newOptions(opts ...Option) *_Options {
	o := &_Options{
		shutdownSleepDuration: time.Second,
		logInfoFunc: func(msg string, keysAndValues ...interface{}) {
			log.Println(append([]interface{}{"msg", msg}, keysAndValues...)...)
		},
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func WithServerOptions(opts ...grpc.ServerOption) Option {
	return func(o *_Options) {
		o.serverOptions = append(o.serverOptions, opts...)
	}
}

func WithMetadata(metadata []byte) Option {
	return func(o *_Options) {
		o.metadata = metadata
	}
}

func WithShutdownSleepDuration(shutdownSleepDuration time.Duration) Option {
	return func(o *_Options) {
		o.shutdownSleepDuration = shutdownSleepDuration
	}
}

func WithLogInfoFunc(logInfoFunc func(msg string, keysAndValues ...interface{})) Option {
	return func(o *_Options) {
		o.logInfoFunc = logInfoFunc
	}
}
