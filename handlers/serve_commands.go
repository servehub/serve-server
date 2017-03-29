package handlers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	consulApi "github.com/hashicorp/consul/api"
	"github.com/kulikov/go-sbus"

	"github.com/servehub/serve-server/handler"
	"github.com/servehub/utils"
	"github.com/servehub/utils/gabs"
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
	servePath := fmt.Sprintf("%s", conf.Path("serve-path").Data())

	bus.Sub("serve-undeploy", func(cmd sbus.Message) error {
		log.Infoln("Receive", cmd)

		m := &undeployCmd{}
		if err := cmd.Unmarshal(m); err != nil {
			return fmt.Errorf("Error on unmarshal undeployCmd: %v", err)
		}

		pairs, _, err := consul.KV().List(dataPrefix+m.Name+"/deploy.", nil)
		if err != nil {
			return err
		}

		if len(pairs) == 0 {
			return fmt.Errorf("Service data not found for undeploy `%s`!", m.Name)
		}

		for _, item := range pairs {
			// validate and reserealize json without spaces
			js := make(map[string]interface{})
			if err := json.Unmarshal(item.Value, &js); err != nil {
				return fmt.Errorf("Error on parse deploy data: %v", err)
			}
			js["purge"] = true
			str, _ := json.Marshal(js)
			pluginData := strings.Replace(string(str), "'", "\\'", -1)

			names := strings.Split(item.Key, "/")
			if err := utils.RunCmd("%s %s --plugin-data='%s'", servePath, names[len(names)-1], pluginData); err != nil {
				return err
			}
		}

		bus.Reply(cmd, nil)
		return nil
	})

	return nil
}

type undeployCmd struct {
	Name string `json:"name"`
}
