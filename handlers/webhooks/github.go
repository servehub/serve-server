package webhooks

import (
	"fmt"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/kulikov/go-sbus"

	"github.com/servehub/utils"
	"github.com/servehub/utils/gabs"

	"github.com/servehub/serve-server/handler"
	"github.com/servehub/serve-server/models"
)

func init() {
	handler.HandlerRegestry.Add("webhooks-github", &WebhooksGithub{})
}

type WebhooksGithub struct{}

func (_ WebhooksGithub) Run(bus *sbus.Sbus, conf *gabs.Container, log *logrus.Entry) error {

	bus.Sub("receive-webhook-github", func(msg sbus.Message) error {
		data, err := gabs.ParseJSON(msg.Data)
		if err != nil {
			return err
		}

		log.Debugln("Receive webhook: ", data.StringIndent("", "  "))

		repo := data.Path("repository.ssh_url").String()
		branch := strings.TrimPrefix(data.Path("ref").String(), "refs/heads/")
		closed := "true" == fmt.Sprintf("%v", data.Path("deleted").Data())

		tmp := fmt.Sprintf("/tmp/serve/manifests/%s/%s", data.Path("repository.full_name").Data(), branch)
		if err := os.MkdirAll(tmp, os.ModePerm); err != nil {
			return err
		}

		manifest := tmp + "/manifest.yml"
		oldHash := md5check(manifest)

		if !closed {
			if err := utils.RunCmd(
				"git archive --remote=%s %s manifest.yml | tar -xC %s",
				repo,
				branch,
				tmp,
			); err != nil {
				if err.Error() == "exit status 1" {
					log.Debugf("manifest.yml not found in `%s`!", repo)
					return nil
				} else {
					return err
				}
			}
		}

		if closed || oldHash != md5check(manifest) {
			return bus.Pub("manifest-changed", models.ManifestChanged{
				Manifest: manifest,
				Repo:     repo,
				Branch:   branch,
				Purge:    closed,
			})
		} else {
			log.Debugln("Manifest not changed")
		}

		return nil
	})

	return nil
}
