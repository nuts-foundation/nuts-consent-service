package pkg

import (
	"context"
	eh "github.com/looplab/eventhorizon"
	"github.com/sirupsen/logrus"
)

func logger() *logrus.Entry {
	return logrus.StandardLogger().WithField("module", "consent-service")
}

type EventLogger struct{}

func (e EventLogger) HandlerType() eh.EventHandlerType {
	return eh.EventHandlerType("EventLogger")
}

func (e EventLogger) HandleEvent(ctx context.Context, event eh.Event) error {
	logger().Debugf("[Eventlogger]: %+v \n", event)
	return nil
}

func (e EventLogger) CommandLogger(h eh.CommandHandler) eh.CommandHandler {
	return eh.CommandHandlerFunc(func(ctx context.Context, command eh.Command) error {
		logger().Debugf("CMD %#v", command)
		return h.HandleCommand(ctx, command)
	})
}

