package puppet

import (
	"context"
	"log"
	"os"
	"os/exec"
	"ptr/pkg/contract"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Puppet struct {
	Name          string
	ServerAddress string
}

func New(name, serverAddress string) *Puppet {
	return &Puppet{
		Name:          name,
		ServerAddress: serverAddress,
	}
}

func (p *Puppet) Start() error {
	conn, err := grpc.NewClient(p.ServerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	client := contract.NewPuppetServiceClient(conn)

	ctx := context.Background()

	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		stream, err := client.GetCommands(egCtx, &contract.GetCommandRequest{
			PuppetName: p.Name,
		})
		if err != nil {
			return err
		}

		for {
			cmd, err := stream.Recv()
			if err != nil {
				return err
			}

			output, err := exec.Command("/bin/sh", "-c", cmd.Command).CombinedOutput()
			if err != nil {
				return err
			}

			log.Printf("Output of running the requested commmand: %s is \n%s", cmd.Command, output)
			_, err = client.SendResult(egCtx, &contract.SendResultRequest{
				CommandId: cmd.CommandId,
				Result:    string(output),
			})
			if err != nil {
				return err
			}
		}
	})

	eg.Go(func() error {
		stream, err := client.Heartbeat(egCtx, &contract.PingRequest{
			PuppetName: p.Name,
		})
		if err != nil {
			return err
		}

		for {
			_, err := stream.Recv()
			if err != nil {
				return err
			}

			log.Printf("Ping received")
		}
	})

	eg.Go(func() error {
		stream, err := client.GetFile(egCtx, &contract.GetFileRequest{
			PuppetName: p.Name,
		})
		if err != nil {
			return err
		}

		for {
			log.Printf("Waiting for file for %s", p.Name)
			file, err := stream.Recv()
			if err != nil {
				return err
			}

			log.Printf("Received file %s", file.FileName)
			err = os.WriteFile(file.FileName, []byte(file.Contents), 0644)
			if err != nil {
				return err
			}
		}
	})

	return eg.Wait()
}
