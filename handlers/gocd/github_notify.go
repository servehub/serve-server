package gocd

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/kulikov/go-sbus"

	"github.com/servehub/utils/gabs"

	"github.com/servehub/serve-server/handler"
)

func init() {
	handler.HandlerRegestry.Add("gocd-github-notify", &GithubNotify{})
}

type GithubNotify struct{}

var httpClient = &http.Client{Transport: &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}}

func (_ GithubNotify) Run(bus *sbus.Sbus, conf *gabs.Container, log *logrus.Entry) error {

	bus.Sub("receive-webhook-gocd-github-notify", func(msg sbus.Message) error {
		req, _ := http.NewRequest("POST", fmt.Sprintf("%s", conf.Path("notify-url").Data()), bytes.NewReader(msg.Meta["body"].([]byte)))
		req.Header.Set("Content-type", "application/json")

		for h, v := range msg.Meta["headers"].(http.Header) {
			if strings.HasPrefix(strings.ToLower(h), "x-github-") || strings.HasPrefix(strings.ToLower(h), "x-hub-") {
				req.Header.Set(h, v[0])
			}
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return err
		}

		defer resp.Body.Close()
		respBody, _ := ioutil.ReadAll(resp.Body)

		log.Printf(" <-- %s\n%s", resp.Status, string(respBody))
		return nil
	})

	return nil
}
