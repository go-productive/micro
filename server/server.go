package server

import (
	"fmt"
	"github.com/go-productive/micro/registry"
	"google.golang.org/grpc"
	"net"
	"strings"
	"time"
)

type (
	GRPCServer struct {
		server   *grpc.Server
		registry registry.Registry
		options  *_Options

		addr               string
		deregisterFunc     []func()
		serviceNameMapNode map[string]*registry.Node
	}
)

func New(addr string, registry registry.Registry, opts ...Option) *GRPCServer {
	options := newOptions(opts...)
	server := grpc.NewServer(options.serverOptions...)
	g := &GRPCServer{
		server:   server,
		registry: registry,
		options:  options,
		addr:     addr,
	}
	g.initRegistryAddr()
	return g
}

func (g *GRPCServer) initRegistryAddr() {
	split := strings.Split(g.addr, ":")
	if split[0] == "" {
		ip, err := privateIP()
		if err != nil {
			panic(err)
		}
		g.addr = fmt.Sprintf("%v:%v", ip, split[1])
	}
}

func (g *GRPCServer) RegistryAddr() string {
	return g.addr
}

func (g *GRPCServer) Nodes() map[string]*registry.Node {
	return g.serviceNameMapNode
}

func (g *GRPCServer) GRPCServer() *grpc.Server {
	return g.server
}

func (g *GRPCServer) Serve() error {
	listener, err := net.Listen("tcp", g.addr)
	if err != nil {
		return err
	}
	defer listener.Close()
	if err := g.register(); err != nil {
		return err
	}
	return g.server.Serve(listener)
}

func (g *GRPCServer) GracefulStop() {
	g.deregister()
	time.Sleep(g.options.shutdownSleepDuration)
	g.server.GracefulStop()
}

func (g *GRPCServer) register() (err error) {
	defer func() {
		if err != nil {
			g.deregister()
		}
	}()
	g.serviceNameMapNode = make(map[string]*registry.Node, len(g.server.GetServiceInfo()))
	for serviceName := range g.server.GetServiceInfo() {
		node := &registry.Node{
			ServiceName: serviceName,
			Addr:        g.addr,
			Metadata:    g.options.metadata,
		}
		deregisterFunc, err := g.registry.Register(node)
		if err != nil {
			return err
		}
		g.serviceNameMapNode[serviceName] = node
		g.options.logInfoFunc("Register", "node", node)
		g.deregisterFunc = append(g.deregisterFunc, func() {
			deregisterFunc()
			g.options.logInfoFunc("Deregister", "node", node)
		})
	}
	return nil
}

func (g *GRPCServer) deregister() {
	for _, f := range g.deregisterFunc {
		f()
	}
}
