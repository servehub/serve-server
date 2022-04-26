package gocd

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/kulikov/go-sbus"

	"github.com/servehub/utils/gabs"

	"github.com/servehub/serve-server/handler"
	"github.com/servehub/serve-server/models"
)

func init() {
	handler.HandlerRegestry.Add("gocd-run-pipeline", &RunPipeline{})
}

type RunPipeline struct{}

func (_ RunPipeline) Run(bus *sbus.Sbus, conf *gabs.Container, log *logrus.Entry) error {

	bus.Sub("github-code-updated", func(cmd sbus.Message) error {
		update := &models.CodeUpdated{}
		if err := cmd.Unmarshal(update); err != nil {
			return fmt.Errorf("Error on unmarshal github-code-updated: %v", err)
		}

		if update.Commit == "0000000000000000000000000000000000000000" {
			update.Commit = "master"
		}

		gocdUrl := fmt.Sprintf("%s", conf.Path("gocd-url").Data())
		body := fmt.Sprintf(`{"environment_variables": [{"name": "BRANCH", "value": "%s"}, {"name": "COMMIT", "value": "%s"}, {"name": "PREVIOUS_COMMIT", "value": "%s"}], "update_materials_before_scheduling": true}`, update.Branch, update.Commit, update.PrevCommit)

		pipelines, _ := conf.Path("pipelines").Children()

		for _, pipeline := range pipelines {
			if pipeline.Path("repo").Data() == update.Repo {
				goCdRequest("POST", fmt.Sprintf("%s/go/api/pipelines/%s/schedule", gocdUrl, pipeline.Path("pipeline").Data()), body, map[string]string{"Accept": "application/vnd.go.cd.v1+json"})
			}
		}

		return nil
	})

	return nil
}

type goCdCredents struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

var httpClient = &http.Client{Transport: &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}}

var goCdRequest = func(method string, resource string, body string, headers map[string]string) (*http.Response, error) {
	req, _ := http.NewRequest(method, resource, bytes.NewReader([]byte(body)))

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	req.Header.Set("Content-Type", "application/json")

	data, err := ioutil.ReadFile("/etc/serve/gocd_credentials")
	if err != nil {
		return nil, fmt.Errorf("Credentias file error: %v", err)
	}

	creds := &goCdCredents{}
	json.Unmarshal(data, creds)

	req.SetBasicAuth(creds.Login, creds.Password)

	log.Printf(" --> %s %s:\n%s\n%s\n\n", method, resource, req.Header, body)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)

	log.Printf("<-- %s\n%s", resp.Status, string(respBody))
	return resp, nil
}
