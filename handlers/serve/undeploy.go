package serve

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/kulikov/go-sbus"

	"github.com/servehub/serve-server/handler"
	"github.com/servehub/serve-server/models"
	"github.com/servehub/utils"
	"github.com/servehub/utils/gabs"
)

func init() {
	handler.HandlerRegestry.Add("serve-undeploy-service", &UndeployService{})
}

/**
 * Undeploy service after branch remove
 */
type UndeployService struct{}

func (_ UndeployService) Run(bus *sbus.Sbus, conf *gabs.Container, log *logrus.Entry) error {
	bus.Sub("manifest-changed", func(cmd sbus.Message) error {
		m := &models.ManifestChanged{}
		if err := cmd.Unmarshal(m); err != nil {
			return fmt.Errorf("Error on unmarshal manifestChanged: %v", err)
		}

		if m.Purge {
			return utils.RunCmd(
				"serve outdated --manifest=%s --env=%s --zone=%s --branch=%s",
				m.Manifest,
				conf.Path("env").Data(),
				conf.Path("zone").Data(),
				m.Branch,
			)
		}

		return nil
	})

	return nil
}
