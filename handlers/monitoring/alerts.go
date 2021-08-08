package gocd

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/kulikov/go-sbus"

	"github.com/servehub/utils"
	"github.com/servehub/utils/gabs"

	"github.com/servehub/serve-server/handler"
	"github.com/servehub/serve-server/models"
)

func init() {
	handler.HandlerRegestry.Add("update-monitoring-alerts", &UpdateMonitoringAlerts{})
}

type UpdateMonitoringAlerts struct{}

/**
 * Update monitoring alert in Consul
 */
func (_ UpdateMonitoringAlerts) Run(bus *sbus.Sbus, conf *gabs.Container, log *logrus.Entry) error {

	bus.Sub("manifest-changed", func(cmd sbus.Message) error {
		m := &models.ManifestChanged{}
		if err := cmd.Unmarshal(m); err != nil {
			return fmt.Errorf("Error on unmarshal manifestChanged: %v", err)
		}

		if m.Branch != "master" {
			return nil
		}

		return utils.RunCmd(
			"serve monitoring --manifest=%s --env=%s --zone=%s",
			m.Manifest,
			conf.Path("env").Data(),
			conf.Path("zone").Data(),
		)
	})

	return nil
}
