package main

import (
	"context"
	"github.com/go-productive/micro/example/internal"
	"github.com/go-productive/micro/registry/etcdv3"
	"github.com/go-productive/micro/server"
)

type (
	EchoServer struct {
	}
)

func (e *EchoServer) Echo(ctx context.Context, req *internal.EchoMsg) (*internal.EchoMsg, error) {
	return req, nil
}

func main() {
	discoveryRegistry := etcdv3.New([]string{"192.168.42.141:13579"})
	grpcServer := server.New(":64000", discoveryRegistry)
	internal.RegisterEchoServer(grpcServer.GRPCServer(), new(EchoServer))
	panic(grpcServer.Serve())
}
