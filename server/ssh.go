package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"golang.org/x/crypto/ssh"

	"github.com/trunners/runners/logger"
)

// pipe SSH channel from server to client.
func pipe(ctx context.Context, channel ssh.NewChannel, client *ssh.Client) error {
	log := logger.FromContext(ctx)

	if t := channel.ChannelType(); t != "session" {
		err := channel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
		if err != nil {
			log.WarnContext(ctx, "Could not reject channel", "error", err)
		}

		return errors.New("unknown channel type")
	}

	serverChannel, serverReqs, err := channel.Accept()
	if err != nil {
		log.ErrorContext(ctx, "Could not accept channel", "error", err)
		return err
	}

	clientChannel, clientReqs, err := client.OpenChannel("session", nil)
	if err != nil {
		log.ErrorContext(ctx, "Could not create ssh session", "error", err)
		return err
	}

	// Cleanup function
	cleanup := func() {
		err = clientChannel.Close()
		if err != nil {
			log.WarnContext(ctx, "Could not close client", "error", err)
		}

		err = serverChannel.Close()
		if err != nil {
			log.WarnContext(ctx, "Could not close server", "error", err)
		}
	}

	// Pipe channels
	wg := sync.WaitGroup{}
	once := sync.Once{}

	wg.Go(func() {
		_, err = io.Copy(serverChannel, clientChannel)
		if err != nil {
			log.WarnContext(ctx, "Error copying from server to client", "error", err)
		}

		once.Do(cleanup)
	})

	wg.Go(func() {
		_, err = io.Copy(clientChannel, serverChannel)
		if err != nil {
			log.WarnContext(ctx, "Error copying from client to server", "error", err)
		}

		once.Do(cleanup)
	})

	wg.Go(func() {
		request(ctx, clientChannel, serverReqs)
	})

	wg.Go(func() {
		request(ctx, serverChannel, clientReqs)
	})

	wg.Wait()
	return nil
}

// request forwards SSH requests between server and client channels.
func request(ctx context.Context, channel ssh.Channel, requests <-chan *ssh.Request) {
	log := logger.FromContext(ctx)

	for req := range requests {
		log.DebugContext(ctx, "Sending request", "type", req.Type)

		var reply bool
		reply, err := channel.SendRequest(req.Type, req.WantReply, req.Payload)
		if err != nil {
			log.ErrorContext(ctx, "Error sending request to client", "error", err)
			continue
		}

		if req.WantReply {
			err = req.Reply(reply, nil)
			if err != nil {
				log.ErrorContext(ctx, "Error replying to server request", "error", err)
			}
		}
	}
}
