package consul

import (
	"encoding/json"
	"fmt"
	"github.com/servehub/serve-server/models"
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
	handler.HandlerRegestry.Add("consul-remove-outdated", &ConsulRemoveOutdated{})
}

type ConsulRemoveOutdated struct{}

//
// Periodically look for an entry with an outdated service in consul kv,
// if find it and the endOfLife time has come, delete this service
//
func (_ ConsulRemoveOutdated) Run(bus *sbus.Sbus, conf *gabs.Container, log *logrus.Entry) error {
	cf := consulApi.DefaultConfig()
	cf.Address = fmt.Sprintf("%s", conf.Path("consul").Data())
	consul, err := consulApi.NewClient(cf)
	if err != nil {
		return err
	}

	outdatedPrefix := fmt.Sprintf("%s", conf.Path("outdated-prefix").Data())
	dataPrefix := fmt.Sprintf("%s", conf.Path("service-data-prefix").Data())
	checkInterval, err := time.ParseDuration(fmt.Sprintf("%s", conf.Path("check-interval").Data()))
	if err != nil {
		return err
	}

	log.Infof("Connecting to consul://%s", cf.Address)

	go func() {
		for range time.Tick(checkInterval) {
			pairs, _, err := consul.KV().List(outdatedPrefix, nil)
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
					name := strings.TrimPrefix(item.Key, outdatedPrefix)
					key := item.Key
					log.WithField("json", string(item.Value)).Infoln("Found outdated service:", name)

					if err := undeployService(name, dataPrefix, consul, log); err == nil {
						log.Infof("Service `%s` deleted! Remove outdated key `%s`...", name, key)
						utils.DelConsulKv(consul, key)
					} else {
						log.Warnf("Error on undeploing service `%s`: %v", name, err)
					}
				}
			}
		}
	}()

	if conf.Exists("cleanup-branches") && conf.Path("cleanup-branches").Data() == true {

		bus.Sub("manifest-changed", func(cmd sbus.Message) error {
			m := &models.ManifestChanged{}
			if err := cmd.Unmarshal(m); err != nil {
				return fmt.Errorf("Error on unmarshal manifestChanged: %v", err)
			}

			if !m.Purge {
				return nil
			}

			pairs, _, err := consul.KV().List(dataPrefix, nil)
			if err != nil {
				return err
			}

			for _, item := range pairs {
				if !strings.HasSuffix(item.Key, "deploy.marathon") {
					continue
				}

				data := string(item.Value)

				if !strings.Contains(data, m.Repo) || !strings.Contains(data, m.Branch) {
					continue
				}

				deploy := &models.ServiceDeployData{}
				if err := json.Unmarshal(item.Value, deploy); err != nil {
					log.Errorf("Error on unmarshal ServiceDeployData: %v", err)
					continue
				}

				log.Debugln("Removed service:", item.Key, deploy)

				if deploy.GitRepo != m.Repo || deploy.Branch != m.Branch {
					continue
				}

				if err := utils.MarkAsOutdated(consul, deploy.AppName, 0); err != nil {
					log.Errorf("Error on MarkAsOutdated: %v", err)
				}

				if err := utils.DelConsulKv(consul, "services/routes/"+deploy.AppName); err != nil {
					log.Errorf("Error on DelConsulKv: %v", err)
				}
			}

			return nil
		})
	}

	return nil
}

type outdatedService struct {
	EndOfLife string `json:"endOfLife"` // todo: change EndOfLife type to string in serve.release plugin
}

func undeployService(name string, dataPrefix string, consul *consulApi.Client, log *logrus.Entry) error {
	pairs, _, err := consul.KV().List(dataPrefix+name+"/deploy.", nil)
	if err != nil {
		return err
	}

	var outputError error

	if len(pairs) > 0 {
		for _, item := range pairs {
			// validate and reserealize json without spaces
			js := make(map[string]interface{})
			if err := json.Unmarshal(item.Value, &js); err != nil {
				outputError = fmt.Errorf("Error on parse deploy data: %v", err)
				continue
			}

			js["purge"] = true
			str, _ := json.Marshal(js)

			err = utils.WriteTemp(str, func(filePath string) error {
				names := strings.Split(item.Key, "/")
				return utils.RunCmd("serve %s --plugin-data='%s'", names[len(names)-1], filePath)
			})

			if err != nil {
				log.Warnf("Service `%s` undeploy %s with error: %v", name, item.Key, err)
				outputError = err
			}
		}
	} else {
		log.Warnf("Service data not found for undeploy `%s`! Skip...", name)
	}

	return outputError
}
