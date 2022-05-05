package webhooks

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
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

	xhubSecret := []byte(fmt.Sprintf("%s", conf.Path("xhub-secret").Data()))

	bus.Sub("receive-webhook-github", func(msg sbus.Message) error {
		data, err := gabs.ParseJSON(msg.Data)
		if err != nil {
			return err
		}

		log.Debugln("Receive webhook:", msg.Subject)

		if len(xhubSecret) > 0 {
			xpub := fmt.Sprintf("%s", msg.Meta["headers"].(http.Header).Get("X-Hub-Signature"))
			mac := hmac.New(sha1.New, xhubSecret)
			mac.Write(msg.Meta["body"].([]byte))

			expected := "sha1=" + hex.EncodeToString(mac.Sum(nil))
			if xpub != expected {
				return fmt.Errorf("X-Hub-Signature not valid! Expected '" + expected + "', given '" + xpub + "'")
			} else {
				log.Info("X-Hub-Signature is valid!")
			}
		}

		repo := fmt.Sprintf("%s", data.Path("repository.ssh_url").Data())
		branch := strings.TrimPrefix(fmt.Sprintf("%s", data.Path("ref").Data()), "refs/heads/")
		fullName := fmt.Sprintf("%s", data.Path("repository.full_name").Data())
		closed := "true" == fmt.Sprintf("%v", data.Path("deleted").Data())

		tmp := fmt.Sprintf("/tmp/serve/manifests/%s/%s", fullName, branch)
		if err := os.MkdirAll(tmp, os.ModePerm); err != nil {
			return err
		}

		manifest := tmp + "/manifest.yml"
		oldHash := md5check(manifest)

		bus.Pub("github-code-updated", models.CodeUpdated{
			Repo:       repo,
			Branch:     branch,
			Commit:     fmt.Sprintf("%s", data.Path("after").Data()),
			PrevCommit: fmt.Sprintf("%s", data.Path("before").Data()),
		})

		if !closed {
			fileUrl := fmt.Sprintf("https://api.github.com/repos/%s/contents/manifest.yml?ref=%s", fullName, branch)
			req, _ := http.NewRequest("GET", fileUrl, nil)

			log.Debug("Request manifest.yml: " + fileUrl)

			req.Header.Set("Authorization", fmt.Sprintf("token %s", conf.Path("token").Data()))
			req.Header.Set("Accept", "application/vnd.github.v3.raw")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 {
				log.Debugf("manifest.yml not found in `%s`!", repo)
				return nil
			} else if resp.StatusCode != 200 {
				return errors.New(resp.Status)
			}

			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if err := ioutil.WriteFile(manifest, body, 0644); err != nil {
				return err
			}
		} else {
			if _, err := os.Stat(manifest); !os.IsNotExist(err) {
				utils.RunCmd("echo '\n # deleted' >> %s", manifest)
			}
		}

		newHash := md5check(manifest)

		if newHash != "" && (closed || oldHash != newHash) {
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
