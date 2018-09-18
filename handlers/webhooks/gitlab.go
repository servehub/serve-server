package webhooks

import (
	"fmt"
	"github.com/servehub/utils"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/kulikov/go-sbus"

	"github.com/servehub/utils/gabs"

	"github.com/servehub/serve-server/handler"
	"github.com/servehub/serve-server/models"
)

func init() {
	handler.HandlerRegestry.Add("webhooks-gitlab", &WebhooksGitlab{})
}

type WebhooksGitlab struct{}

func (_ WebhooksGitlab) Run(bus *sbus.Sbus, conf *gabs.Container, log *logrus.Entry) error {

	bus.Sub("receive-webhook-gitlab", func(msg sbus.Message) error {
		data, err := gabs.ParseJSON(msg.Data)
		if err != nil {
			return err
		}

		log.Debugln("Receive webhook: ", data.StringIndent("", "  "))

		repo := fmt.Sprintf("%s", data.Path("project.ssh_url").Data())
		branch := strings.TrimPrefix(fmt.Sprintf("%s", data.Path("changes.ref").Data()), "refs/heads/")
		fullName := fmt.Sprintf("%s", data.Path("project.name").Data())
		closed := "0000000000000000000000000000000000000000" == fmt.Sprintf("%v", data.Path("changes.after").Data())

		tmp := fmt.Sprintf("/tmp/serve/manifests/%s/%s", fullName, branch)
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
