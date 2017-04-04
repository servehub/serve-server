package handlers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	consulApi "github.com/hashicorp/consul/api"
	"github.com/kulikov/go-sbus"

	"github.com/servehub/serve-server/handler"
	"github.com/servehub/utils"
	"github.com/servehub/utils/gabs"
)

func init() {
	handler.HandlerRegestry.Add("consul-check-outdated", &ConsulCheckOutdated{})
}

type ConsulCheckOutdated struct{}

// Переодически ищем в consul kv запись с outdated сервисом,
// если находим и время endOfLife пришло — удаляем этот сервис
func (_ ConsulCheckOutdated) Run(bus *sbus.Sbus, conf *gabs.Container, log *logrus.Entry) error {
	cf := consulApi.DefaultConfig()
	cf.Address = fmt.Sprintf("%s", conf.Path("consul").Data())
	consul, err := consulApi.NewClient(cf)
	if err != nil {
		return err
	}

	keyPrefix := fmt.Sprintf("%s", conf.Path("prefix").Data())
	checkInterval, err := time.ParseDuration(fmt.Sprintf("%s", conf.Path("check-interval").Data()))
	if err != nil {
		return err
	}

	log.Infof("Connecting to consul://%s", cf.Address)

	for range time.Tick(checkInterval) {
		pairs, _, err := consul.KV().List(keyPrefix, nil)
		if err != nil {
			log.WithError(err).Error("Error on list outdated on consul!")
			continue
		}

		for _, item := range pairs {
			log.Debugln("Outdated service:", item.Key, string(item.Value))

			outdated := &outdatedService{}
			err := json.Unmarshal(item.Value, outdated)
			if err != nil {
				log.WithError(err).WithField("json", string(item.Value)).Error("Error on parse outdated json!")
				continue
			}

			endOfLife, err := time.Parse(time.RFC3339, outdated.EndOfLife)
			if err != nil {
				log.WithError(err).Errorln("Error on parse `endOfLife` time:", outdated.EndOfLife)
				continue
			}

			if endOfLife.Unix() < time.Now().Unix() {
				name := strings.TrimPrefix(item.Key, keyPrefix)
				log.WithField("json", string(item.Value)).Infoln("Found outdated service:", name)

				bus.Request("serve-undeploy", map[string]string{"name": name}, func(resp sbus.Message) error {
					log.Infof("Service `%s` deleted! Remove outdated key...", name)
					return utils.DelConsulKv(consul, item.Key)
				}, time.Minute*3)
			}
		}
	}

	return nil
}

type outdatedService struct {
	EndOfLife string `json:"endOfLife"` // todo: change EndOfLife type to string in serve.release plugin
}
