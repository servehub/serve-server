package handlers

import (
	"github.com/Sirupsen/logrus"
	"github.com/kulikov/go-sbus"
	consulApi "github.com/hashicorp/consul/api"

	"github.com/servehub/serve-server/handler"
	"github.com/servehub/utils/gabs"
	"fmt"
)

func init() {
	handler.HandlerRegestry.Add("serve-commands", &ServeCommands{})
}

type ServeCommands struct{}

func (_ ServeCommands) Run(bus *sbus.Sbus, conf *gabs.Container, log *logrus.Entry) error {
	cf := consulApi.DefaultConfig()
	cf.Address = fmt.Sprintf("%s", conf.Path("consul").Data())
	consul, err := consulApi.NewClient(cf)
	if err != nil {
		return err
	}

	dataPrefix := fmt.Sprintf("%s", conf.Path("service-data-prefix").Data())

	bus.Sub("serve-undeploy-service", func(cmd sbus.Message) error {
		log.Infoln("Receive", cmd)

		pairs, _, err := consul.KV().List(dataPrefix, nil)
		if err != nil {
			return err
		}

		for _, item := range pairs {
			log.Info(item)
		}

		bus.Reply(cmd, nil)
		return nil
	})

	return nil
}
