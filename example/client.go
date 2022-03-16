package main

import (
	"fmt"
	"github.com/go-productive/micro/client"
	"github.com/go-productive/micro/client/selector"
	"github.com/go-productive/micro/example/internal"
	"github.com/go-productive/micro/registry/etcdv3"
	"time"
)

func main() {
	discoveryRegistry := etcdv3.New([]string{"192.168.42.141:13579"})
	grpcClient := client.New(discoveryRegistry)
	echoClient := internal.NewEchoClient(grpcClient.ClientConn())
	fmt.Println(echoClient.Echo(selector.WithRoundRobin(nil), &internal.EchoMsg{
		Msg: time.Now().String(),
	}))
}
