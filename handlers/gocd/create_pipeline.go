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
	handler.HandlerRegestry.Add("gocd-create-pipeline", &GocdCreatePipeline{})
}

type GocdCreatePipeline struct{}

func (_ GocdCreatePipeline) Run(bus *sbus.Sbus, conf *gabs.Container, log *logrus.Entry) error {

	bus.Sub("manifest-changed", func(cmd sbus.Message) error {
		m := &models.ManifestChanged{}
		if err := cmd.Unmarshal(m); err != nil {
			return fmt.Errorf("Error on unmarshal manifestChanged: %v", err)
		}

		return utils.RunCmd(
			"serve gocd.pipeline.create --manifest=%s --ssh-repo=%s --branch=%s --purge=%v",
			m.Manifest,
			m.Repo,
			m.Branch,
			m.Purge,
		)
	})

	return nil
}
