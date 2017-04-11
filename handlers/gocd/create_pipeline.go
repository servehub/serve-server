package gocd

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/kulikov/go-sbus"

	"github.com/servehub/serve-server/handler"
	"github.com/servehub/utils"
	"github.com/servehub/utils/gabs"
)

func init() {
	handler.HandlerRegestry.Add("gocd-create-pipeline", &GocdCreatePipeline{})
}

type GocdCreatePipeline struct{}

func (_ GocdCreatePipeline) Run(bus *sbus.Sbus, conf *gabs.Container, log *logrus.Entry) error {

	bus.Sub("manifest-changed", func(cmd sbus.Message) error {
		m := &manifestChanged{}
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

type manifestChanged struct {
	Manifest string `json:"manifest"`
	Repo     string `json:"repo"`
	Branch   string `json:"branch"`
	Purge    bool   `json:"purge,string"`
}
