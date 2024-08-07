package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"ptr/pkg/cache"
	"ptr/pkg/contract"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type server struct {
	contract.UnimplementedPuppetServiceServer
	contract.UnimplementedMasterServiceServer
	port  int
	cache cache.Cache
}

func New(port int, cache cache.Cache) *server {
	return &server{
		port:  port,
		cache: cache,
	}
}

func (s *server) List(ctx context.Context, req *contract.ListRequest) (*contract.ListResponse, error) {
	puppets, err := s.cache.ListPuppets(ctx)
	if err != nil {
		return nil, err
	}

	return &contract.ListResponse{
		Puppets: puppets,
	}, nil

}

func (s *server) Heartbeat(req *contract.PingRequest, stream grpc.ServerStreamingServer[contract.PingResponse]) error {
	puppet := req.PuppetName
	ctx := context.Background()

	for {
		err := stream.Send(&contract.PingResponse{})
		if err != nil {
			err = s.cache.RemovePuppet(ctx, puppet)
			if err != nil {
				log.Printf("Unable to remove puppet from list: %s", puppet)
			}

			break
		}

		time.Sleep(10 * time.Second)
	}

	return nil
}

func (s *server) Run(ctx context.Context, req *contract.RunCommandRequest) (*contract.RunCommandResponse, error) {
	u := uuid.New()
	err := s.cache.AddCommand(ctx, req.PuppetName, u.String(), req.Command)
	if err != nil {
		return nil, err
	}

	response, err := s.cache.WaitForCommandResponse(ctx, u.String())
	if err != nil {
		return nil, err
	}

	return &contract.RunCommandResponse{
		Result: response,
	}, nil
}

func (s *server) Copy(ctx context.Context, cr *contract.CopyRequest) (*contract.CopyResponse, error) {
	log.Printf("Copying %s to %s", cr.FileName, cr.PuppetName)
	err := s.cache.WriteFile(ctx, cr.PuppetName, cr.FileName, cr.Contents)
	if err != nil {
		return nil, err
	}

	return &contract.CopyResponse{}, nil
}

func (s *server) GetCommands(req *contract.GetCommandRequest, stream grpc.ServerStreamingServer[contract.Command]) error {

	ctx := context.Background()
	err := s.cache.AddPuppet(ctx, req.PuppetName)
	if err != nil {
		return err
	}
	log.Printf("Added puppet %s to list", req.PuppetName)

	for {
		log.Printf("Waiting for command for %s", req.PuppetName)
		commandID, command, err := s.cache.WaitForCommand(ctx, req.PuppetName)
		if err != nil {
			return err
		}

		log.Printf("Sending command %s to %s", commandID, req.PuppetName)
		err = stream.Send(&contract.Command{
			Command:   command,
			CommandId: commandID,
		})
		if err != nil {
			s.cache.RemovePuppet(ctx, req.PuppetName)
			return err
		}
	}

}
func (s *server) SendResult(ctx context.Context, req *contract.SendResultRequest) (*contract.SendResultResponse, error) {
	err := s.cache.AddCommandResponse(ctx, req.CommandId, req.Result)
	if err != nil {
		return nil, nil
	}
	return &contract.SendResultResponse{}, nil
}

func (s *server) GetFile(req *contract.GetFileRequest, stream grpc.ServerStreamingServer[contract.File]) error {
	ctx := context.Background()

	for {
		log.Printf("Waiting for file for %s", req.PuppetName)
		filename, contents, err := s.cache.WaitForFile(ctx, req.PuppetName)
		if err != nil {
			return err
		}

		log.Printf("Sending file %s to %s", filename, req.PuppetName)
		err = stream.Send(&contract.File{
			FileName: filename,
			Contents: contents,
		})
		if err != nil {
			return err
		}
	}
}

func (s *server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", s.port))
	if err != nil {
		return err
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	contract.RegisterPuppetServiceServer(grpcServer, s)
	contract.RegisterMasterServiceServer(grpcServer, s)
	err = grpcServer.Serve(lis)
	if err != nil {
		return err
	}

	return nil
}
