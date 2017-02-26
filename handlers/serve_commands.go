package handlers

import (
	"github.com/Sirupsen/logrus"
	"github.com/kulikov/go-sbus"

	"github.com/servehub/serve-server/handler"
	"github.com/servehub/utils/gabs"
)

func init() {
	handler.HandlerRegestry.Add("serve-commands", &ServeCommands{})
}

type ServeCommands struct{}

func (_ ServeCommands) Run(bus *sbus.Sbus, conf *gabs.Container, log *logrus.Entry) error {

	bus.Sub("serve-undeploy-service", func(cmd sbus.Message) error {
		log.Infoln("Receive", cmd)
		bus.Reply(cmd, nil)
		return nil
	})

	return nil
}
