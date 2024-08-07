package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"ptr/pkg/cache"
	"ptr/pkg/contract"
	"ptr/pkg/puppet"
	"ptr/pkg/server"

	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	app := &cli.App{
		Name:  "Puppeteer",
		Usage: "Manage remotely deployed Puppeteer agents (puppets)",
		Commands: []*cli.Command{
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "List puppets",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "server-address",
						Aliases:  []string{"addr"},
						Usage:    "Server address",
						Required: true,
						Value:    "localhost:50051",
					},
				},
				Action: func(cCtx *cli.Context) error {
					client, err := getCliGrpcClient(cCtx.String("server-address"))
					if err != nil {
						return err
					}

					ctx := context.Background()
					res, err := client.List(ctx, &contract.ListRequest{})
					if err != nil {
						return err
					}

					headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
					columnFmt := color.New(color.FgYellow).SprintfFunc()

					tbl := table.New("Name")
					tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

					for _, p := range res.Puppets {
						tbl.AddRow(p)
					}

					tbl.Print()
					return nil
				},
			},
			{
				Name: "run",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "server-address",
						Aliases:  []string{"addr"},
						Usage:    "Server address",
						Required: true,
						Value:    "localhost:50051",
					},
					&cli.StringFlag{
						Name:     "puppet-name",
						Aliases:  []string{"p"},
						Usage:    "Puppet name",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "command",
						Aliases:  []string{"c"},
						Usage:    "Command to run",
						Required: true,
					},
				},
				Usage: "Run a command",
				Action: func(cCtx *cli.Context) error {
					client, err := getCliGrpcClient(cCtx.String("server-address"))
					if err != nil {
						return err
					}

					ctx := context.Background()
					output, err := client.Run(ctx, &contract.RunCommandRequest{
						PuppetName: cCtx.String("puppet-name"),
						Command:    cCtx.String("command"),
					})
					if err != nil {
						return err
					}

					fmt.Println(output.Result)

					return nil
				},
			},
			{
				Name:    "copy",
				Aliases: []string{"cp"},
				Usage:   "Copy artifacts to puppets",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "server-address",
						Aliases: []string{"addr"},
						Usage:   "Server address",
						Value:   "localhost:50051",
					},
					&cli.StringFlag{
						Name:     "file",
						Aliases:  []string{"f"},
						Usage:    "File path",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "puppet-name",
						Aliases:  []string{"pn"},
						Usage:    "Name of the puppet to send the file to",
						Required: true,
					},
				},
				Action: func(cCtx *cli.Context) error {
					client, err := getCliGrpcClient(cCtx.String("server-address"))
					if err != nil {
						return err
					}

					ctx := context.Background()
					fileBytes, err := os.ReadFile(cCtx.String("file"))
					if err != nil {
						return err
					}

					_, err = client.Copy(ctx, &contract.CopyRequest{
						PuppetName: cCtx.String("puppet-name"),
						FileName:   filepath.Base(cCtx.String("file")),
						Contents:   string(fileBytes),
					})
					if err != nil {
						return err
					}

					fmt.Printf("Request file copied")
					return nil
				},
			},
			{
				Name:  "start",
				Usage: "start services",
				Subcommands: []*cli.Command{
					{
						Name:    "puppetmaster",
						Aliases: []string{"pm"},
						Flags: []cli.Flag{
							&cli.IntFlag{
								Name:    "port",
								Aliases: []string{"p"},
								Value:   50051,
								Usage:   "Port to listen on",
							},
						},
						Usage: "Start server",
						Action: func(ctx *cli.Context) error {
							port := ctx.Int("port")

							cache, err := cache.NewRedis("localhost:6379")
							if err != nil {
								return fmt.Errorf("Error instantiating cache %s", err)
							}

							s := server.New(port, cache)
							err = s.Start()
							return err
						},
					},
					{
						Name:  "puppet",
						Usage: "Start agents",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "address",
								Aliases: []string{"addr"},
								Value:   "localhost:50051",
								Usage:   "Pass the server's address",
							},
							&cli.StringFlag{
								Name:     "name",
								Aliases:  []string{"n"},
								Value:    "",
								Usage:    "Puppet's name",
								Required: true,
							},
						},
						Action: func(ctx *cli.Context) error {
							// This CLI command should be used to start a puppet
							p := puppet.New(ctx.String("name"), ctx.String("address"))
							err := p.Start()
							if err != nil {
								return err
							}

							return nil
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func getCliGrpcClient(serverAddress string) (contract.MasterServiceClient, error) {
	conn, err := grpc.NewClient(serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := contract.NewMasterServiceClient(conn)
	return client, nil
}
