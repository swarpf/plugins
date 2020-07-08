package swarfarm

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	pb "github.com/swarpf/plugins/swarpf-idl/proto-gen-go/proxyapi"
)

// proxy API consumer
type ProxyApiConsumer struct {
	pb.UnimplementedProxyApiConsumerServer
}

func (s *ProxyApiConsumer) OnReceiveApiEvent(_ context.Context, ev *pb.ApiEvent) (*empty.Empty, error) {
	return &empty.Empty{}, OnReceiveApiEvent(ev.GetCommand(), ev.GetRequest(), ev.GetResponse())
}
