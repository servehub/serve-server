package webhooks

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/kulikov/go-sbus"

	"github.com/servehub/serve-server/handler"
	"github.com/servehub/utils"
	"github.com/servehub/utils/gabs"
)

func init() {
	handler.HandlerRegestry.Add("webhooks-bitbucket", &WebhooksBitbucket{})
}

type WebhooksBitbucket struct{}

func (_ WebhooksBitbucket) Run(bus *sbus.Sbus, conf *gabs.Container, log *logrus.Entry) error {

	bus.Sub("receive-webhook-bitbucket", func(msg sbus.Message) error {
		data, err := gabs.ParseJSON(msg.Data)
		if err != nil {
			return err
		}

		log.Debugln("Receive webhook: ", data.StringIndent("", "  "))

		uri, err := url.Parse(fmt.Sprintf("%s", data.Path("repository.links.html.href").Data()))
		if err != nil {
			return err
		}

		closed := fmt.Sprintf("%v", data.Path("push.changes.closed").Data())
		repo := fmt.Sprintf("git@%s:%s.git", uri.Host, data.Path("repository.full_name").Data())
		branch := fmt.Sprintf("%s", data.Path("push.changes.new.name").Data())

		if closed == "true" {
			branch = fmt.Sprintf("%s", data.Path("push.changes.old.name").Data())
		}

		tmp := fmt.Sprintf("/tmp/serve/manifests/%s/%s", data.Path("repository.full_name").Data(), branch)
		if err := os.MkdirAll(tmp, os.ModePerm); err != nil {
			return err
		}

		manifest := tmp + "/manifest.yml"
		oldHash := md5check(manifest)

		if closed != "true" {
			if err := utils.RunCmd(
				"git archive --remote=%s %s manifest.yml | tar -xC %s",
				repo,
				branch,
				tmp,
			); err != nil {
				return err
			}
		}

		if closed == "true" || oldHash != md5check(manifest) {
			return bus.Pub("manifest-changed", map[string]string{
				"manifest": manifest,
				"repo":     repo,
				"branch":   branch,
				"purge":    closed,
			})
		}

		return nil
	})

	return nil
}

func md5check(file string) string {
	f, err := os.Open(file)
	if err != nil {
		return ""
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}