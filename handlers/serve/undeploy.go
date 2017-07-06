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
	handler.HandlerRegestry.Add("serve-undeploy-service", &ServeUndeployService{})
}

type ServeUndeployService struct{}

func (_ ServeUndeployService) Run(bus *sbus.Sbus, conf *gabs.Container, log *logrus.Entry) error {
	bus.Sub("manifest-changed", func(cmd sbus.Message) error {
		m := &models.ManifestChanged{}
		if err := cmd.Unmarshal(m); err != nil {
			return fmt.Errorf("Error on unmarshal manifestChanged: %v", err)
		}

		if m.Purge {
			return utils.RunCmd(
				"serve outdated --manifest=%s --branch=%s",
				m.Manifest,
				m.Branch,
			)
		}

		return nil
	})

	return nil
}
