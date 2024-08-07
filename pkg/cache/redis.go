package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisCache struct {
	connection *redis.Client
}

func NewRedis(address string) (*redisCache, error) {

	rdb := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	return &redisCache{
		connection: rdb,
	}, nil
}

func (r *redisCache) AddPuppet(ctx context.Context, puppetName string) error {
	res := r.connection.SAdd(ctx, "puppets", puppetName)
	if res.Err() != nil {
		return res.Err()
	}

	return nil
}

func (r *redisCache) RemovePuppet(ctx context.Context, puppetName string) error {
	res := r.connection.SRem(ctx, "puppets", puppetName)
	if res.Err() != nil {
		return res.Err()
	}

	return nil
}

func (r *redisCache) ListPuppets(ctx context.Context) ([]string, error) {
	res := r.connection.SMembers(ctx, "puppets")
	return res.Val(), nil
}

func (r *redisCache) AddCommand(ctx context.Context, puppetName, commandID, command string) error {
	commandStr, err := json.Marshal(PuppetCommand{
		CommandId: commandID,
		Command:   command,
	})
	if err != nil {
		return err
	}

	res := r.connection.LPush(ctx, fmt.Sprintf("%s-commands", puppetName), commandStr)
	if res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (r *redisCache) WaitForCommand(ctx context.Context, puppetName string) (string, string, error) {
	log.Printf("Waiting for command for %s", puppetName)
	res := r.connection.BRPop(ctx, time.Hour*2, fmt.Sprintf("%s-commands", puppetName))
	if res.Err() != nil {
		return "", "", res.Err()
	}

	log.Printf("Got command %+v from queue", res.Val()[1])

	pc := PuppetCommand{}
	err := json.Unmarshal([]byte(res.Val()[1]), &pc)
	if err != nil {
		return "", "", res.Err()
	}

	return pc.CommandId, pc.Command, nil
}

func (r *redisCache) AddCommandResponse(ctx context.Context, commandID, response string) error {
	res := r.connection.LPush(ctx, fmt.Sprintf("%s-command-response", commandID), response)
	if res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (r *redisCache) WaitForCommandResponse(ctx context.Context, commandID string) (string, error) {
	res := r.connection.BRPop(ctx, time.Hour*2, fmt.Sprintf("%s-command-response", commandID))
	if res.Err() != nil {
		return "", res.Err()
	}

	return res.Val()[1], nil
}

func (r *redisCache) WriteFile(ctx context.Context, puppetName, fileName, contents string) error {

	fp := fileForPuppet{
		FileName: fileName,
		Contents: contents,
	}
	fileBytes, err := json.Marshal(fp)
	if err != nil {
		return err
	}

	log.Printf("redisCache - Writing file to %s", fmt.Sprintf("%s-files", puppetName))
	res := r.connection.LPush(ctx, fmt.Sprintf("%s-files", puppetName), string(fileBytes))
	if res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (r *redisCache) WaitForFile(ctx context.Context, puppetName string) (string, string, error) {
	res := r.connection.BRPop(ctx, 2*time.Hour, fmt.Sprintf("%s-files", puppetName))
	if res.Err() != nil {
		return "", "", res.Err()
	}

	fp := fileForPuppet{}
	err := json.Unmarshal([]byte(res.Val()[1]), &fp)
	if err != nil {
		return "", "", res.Err()
	}

	return fp.FileName, fp.Contents, nil
}

type PuppetCommand struct {
	CommandId string
	Command   string
}

type PuppetCommandResponse struct {
	CommandId string
	Response  string
}

type fileForPuppet struct {
	FileName string
	Contents string
}
