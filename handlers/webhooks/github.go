package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/cenkalti/backoff"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/kulikov/go-sbus"
	"github.com/servehub/utils"

	"github.com/google/go-github/v44/github"
	"github.com/servehub/serve-server/handler"
	"github.com/servehub/serve-server/models"
	"github.com/servehub/utils/gabs"
	"golang.org/x/oauth2"
)

func init() {
	handler.HandlerRegestry.Add("webhooks-github", &WebhooksGithub{})
}

type WebhooksGithub struct{}

func (_ WebhooksGithub) Run(bus *sbus.Sbus, conf *gabs.Container, log *logrus.Entry) error {

	xhubSecret := []byte(fmt.Sprintf("%s", conf.Path("xhub-secret").Data()))
	githubToken := fmt.Sprintf("%s", conf.Path("token").Data())

	bus.Sub("receive-webhook-github", func(msg sbus.Message) error {
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

		return bus.Pub(fmt.Sprintf("receive-webhook-%s", msg.Meta["headers"].(http.Header).Get("X-GitHub-Event")), msg)
	})

	bus.Sub("receive-webhook-github-push", func(msg sbus.Message) error {
		data, err := gabs.ParseJSON(msg.Data)
		if err != nil {
			return err
		}

		repo := fmt.Sprintf("%s", data.Path("repository.ssh_url").Data())
		branch := strings.TrimPrefix(fmt.Sprintf("%s", data.Path("ref").Data()), "refs/heads/")
		fullName := fmt.Sprintf("%s", data.Path("repository.full_name").Data())
		deleted := "true" == fmt.Sprintf("%v", data.Path("deleted").Data())

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
			Purge:      deleted,
		})

		if !deleted {
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

		if newHash != "" && (deleted || oldHash != newHash) {
			return bus.Pub("manifest-changed", models.ManifestChanged{
				Manifest: manifest,
				Repo:     repo,
				Branch:   branch,
				Purge:    deleted,
			})
		} else {
			log.Debugln("Manifest not changed")
		}

		return nil
	})

	bus.Sub("receive-webhook-github-pull_request", func(msg sbus.Message) error {
		data, err := gabs.ParseJSON(msg.Data)
		if err != nil {
			return err
		}

		title := fmt.Sprintf("%s", data.Path("pull_request.title").Data())

		match, _ := regexp.MatchString("((Revert \")|)(([A-z0-9]{2,10}-\\d+)) .+$", title)

		pullReqState := "failure"
		pullReqDesc := "Failed! Pull request name doesn't follow naming conventions!"

		if match {
			pullReqState = "success"
			pullReqDesc = "Success!"
		}

		if githubToken == "" {
			return errors.New("`GITHUB_TOKEN` is required")
		}

		return SendStatus(githubToken,
			fmt.Sprintf("%s", data.Path("repository.ssh_url").Data()),
			fmt.Sprintf("%s", data.Path("head.sha").Data()),
			pullReqState,
			pullReqDesc,
			"Naming conventions / Pull request",
			"",
		)
	})

	return nil
}

func SendStatus(accessToken string, repo string, ref string, state string, description string, statusContext string, targetUrl string) error {
	rp := strings.SplitN(repo, ":", 2)
	rps := strings.SplitN(rp[1], "/", 2)

	input := &github.RepoStatus{
		State:       github.String(state),
		Description: github.String(description),
		Context:     github.String(statusContext),
	}

	fmt.Printf("%v", input)
	fmt.Printf("%v", ref)

	client := github.NewClient(
		oauth2.NewClient(
			context.Background(),
			oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})))

	return backoff.Retry(func() error {
		_, _, err := client.Repositories.CreateStatus(context.Background(), rps[0], strings.TrimSuffix(rps[1], ".git"), ref, input)
		return err
	}, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3))
}
