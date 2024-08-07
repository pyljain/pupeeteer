package cache

import "context"

type Cache interface {
	AddPuppet(ctx context.Context, puppetName string) error
	RemovePuppet(ctx context.Context, puppetName string) error
	ListPuppets(ctx context.Context) ([]string, error)
	AddCommand(ctx context.Context, puppetName, commandID, command string) error
	WaitForCommand(ctx context.Context, puppetName string) (string, string, error)
	AddCommandResponse(ctx context.Context, commandID, response string) error
	WaitForCommandResponse(ctx context.Context, commandID string) (string, error)
	WriteFile(ctx context.Context, puppetName, fileName, contents string) error
	WaitForFile(ctx context.Context, puppetName string) (string, string, error)
}
