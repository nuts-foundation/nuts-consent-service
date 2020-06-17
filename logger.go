package main

import (
	"context"
	eh "github.com/looplab/eventhorizon"
	"log"
)

type EventLogger struct{}

func (e EventLogger) HandlerType() eh.EventHandlerType {
	return eh.EventHandlerType("EventLogger")
}

func (e EventLogger) HandleEvent(ctx context.Context, event eh.Event) error {
	log.Printf("[Eventlogger]: %+v \n", event)
	return nil
}

func (e EventLogger) CommandLogger(h eh.CommandHandler) eh.CommandHandler {
	return eh.CommandHandlerFunc(func(ctx context.Context, command eh.Command) error {
		log.Printf("CMD %#v", command)
		return h.HandleCommand(ctx, command)
	})
}

