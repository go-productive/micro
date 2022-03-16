package etcdv3

import (
	"log"
	"time"
)

type (
	_Options struct {
		prefix       string
		interval     time.Duration
		ttl          time.Duration
		logErrorFunc func(msg string, keysAndValues ...interface{})
	}
	Option func(*_Options)
)

func newOptions(opts []Option) *_Options {
	o := &_Options{
		prefix:   "/services",
		interval: time.Second * 5,
		logErrorFunc: func(msg string, keysAndValues ...interface{}) {
			log.Println(append([]interface{}{"msg", msg}, keysAndValues...)...)
		},
	}
	o.ttl = o.interval * 3
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func WithPrefix(prefix string) Option {
	return func(o *_Options) {
		o.prefix = prefix
	}
}

func WithIntervalAndTTL(interval, ttl time.Duration) Option {
	return func(o *_Options) {
		o.interval = interval
		o.ttl = ttl
	}
}

func WithLogErrorFunc(logErrorFunc func(msg string, keysAndValues ...interface{})) Option {
	return func(o *_Options) {
		o.logErrorFunc = logErrorFunc
	}
}
