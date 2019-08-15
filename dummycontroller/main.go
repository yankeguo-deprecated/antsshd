package main

import (
	"context"
	"github.com/antssh/types"
	"google.golang.org/grpc"
	"log"
	"net"
)

type dummyController struct{}

func (d *dummyController) Ping(ctx context.Context, req *types.AgentPingReq) (*types.AgentPingResp, error) {
	log.Printf(" PING: %+v", req)
	return &types.AgentPingResp{Success: true, Message: "dummy controller"}, nil
}

func (d *dummyController) Auth(ctx context.Context, req *types.AgentAuthReq) (*types.AgentAuthResp, error) {
	log.Printf("  AUTH: %+v", req)
	return &types.AgentAuthResp{Success: true, Message: "dummy controller"}, nil
}

func (d *dummyController) Record(s types.AgentController_RecordServer) error {
	for {
		f, err := s.Recv()
		if err != nil {
			break
		}
		log.Printf("RECORD: %+v", f)
	}
	return s.SendAndClose(&types.AgentRecordResp{Success: true, Message: "dummy controller"})
}

func main() {
	var err error
	var l net.Listener
	if l, err = net.Listen("tcp", ":2223"); err != nil {
		panic(err)
		return
	}
	s := grpc.NewServer()
	types.RegisterAgentControllerServer(s, &dummyController{})
	if err = s.Serve(l); err != nil {
		panic(err)
	}
}
