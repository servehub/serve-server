package gocd

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/kulikov/go-sbus"

	"github.com/servehub/utils/gabs"

	"github.com/servehub/serve-server/handler"
	"github.com/servehub/serve-server/models"
)

func init() {
	handler.HandlerRegestry.Add("gocd-run-pipeline", &RunPipeline{})
}

var pipelineNameRegex = regexp.MustCompile("\\W+")

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

		if update.PrevCommit == "0000000000000000000000000000000000000000" {
			update.PrevCommit = "master"
		}

		gocdUrl := fmt.Sprintf("%s", conf.Path("gocd-url").Data())

		scheduleBody := fmt.Sprintf(
			`{"environment_variables": [{"name": "BRANCH", "value": "%s"}, {"name": "COMMIT", "value": "%s"}, {"name": "PREVIOUS_COMMIT", "value": "%s"}], "update_materials_before_scheduling": true}`,
			update.Branch,
			update.Commit,
			update.PrevCommit,
		)

		scheduleBodyTree, _ := gabs.ParseJSON([]byte(scheduleBody))
		pipelinesForRun, _ := conf.Path("pipelines").Children()

		for _, pipeline := range pipelinesForRun {
			if pipeline.Path("repo").Data() == update.Repo {

				pipelineName := fmt.Sprintf("%s", pipeline.Path("pipeline").Data())

				if strings.HasPrefix(update.Branch, "feature/") || strings.HasPrefix(update.Branch, "feature-") {
					featureName := strings.ToLower(pipelineNameRegex.ReplaceAllString(strings.TrimPrefix(strings.TrimPrefix(update.Branch, "feature-"), "feature/"), "-"))
					pipelineName = strings.ToLower(pipelineNameRegex.ReplaceAllString(fmt.Sprintf("%s-%s", pipelineName, featureName), "-"))

					exist, _ := goCdRequest("GET", fmt.Sprintf("%s/go/api/admin/pipelines/%s", gocdUrl, pipelineName), "", map[string]string{"Accept": "application/vnd.go.cd.v11+json"})

					if exist.StatusCode == http.StatusNotFound {
						parentResp, err := goCdRequest("GET", fmt.Sprintf("%s/go/api/admin/pipelines/%s", gocdUrl, pipeline.Path("pipeline").Data()), "", map[string]string{"Accept": "application/vnd.go.cd.v11+json"})
						if err != nil {
							return fmt.Errorf("Error on get parent pipeline: %v", err)
						}

						parentBody, _ := ioutil.ReadAll(parentResp.Body)
						parentResp.Body.Close()

						tree, err := gabs.ParseJSON(parentBody)
						if err != nil {
							return fmt.Errorf("Error on parse parent pipeline: %v", err)
						}

						newp := gabs.New()

						tree.Set(pipelineName, "name")
						tree.Set(scheduleBodyTree.Path("environment_variables").Data(), "environment_variables")

						mat, _ := tree.ArrayElementP(0, "materials")
						mat.Set(update.Branch, "attributes", "branch")
						tree.Path("materials").SetIndex(mat.Data(), 0)

						newp.Set(tree.Path("group").Data(), "group")
						newp.Set(tree.Data(), "pipeline")

						err = goCdCreate(pipelineName, "copper", gocdUrl, newp.String(), map[string]string{"Accept": "application/vnd.go.cd.v11+json"})
						if err != nil {
							return fmt.Errorf("Error on create pipeline: %v", err)
						}
					}
				} else if update.Branch != "master" {
					log.Printf("Skip branch %s", update.Branch)
					continue
				}

				goCdRequest("POST", fmt.Sprintf("%s/go/api/pipelines/%s/schedule", gocdUrl, pipelineName), scheduleBody, map[string]string{"Accept": "application/vnd.go.cd.v1+json"})
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

func goCdCreate(name string, env string, resource string, body string, headers map[string]string) error {
	if resp, err := goCdRequest("POST", resource+"/go/api/admin/pipelines", body, headers); err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		if body, err := ioutil.ReadAll(resp.Body); err == nil {
			log.Printf("Operation body: %s", body)
		}
		return fmt.Errorf("Operation error: %s", resp.Status)
	}

	return gocdChangeEnv(resource, env, map[string]interface{}{"add": []string{name}})
}

func goCdUpdate(name string, env string, resource string, body string, headers map[string]string, depends []string) error {
	if resp, err := goCdRequest("PUT", resource+"/go/api/admin/pipelines/"+name, body, headers); err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		if body, err := ioutil.ReadAll(resp.Body); err == nil {
			log.Printf("Operation body: %s", body)
		}
		return fmt.Errorf("Operation error: %s", resp.Status)
	}

	if currentEnv, err := goCdFindEnv(resource, name, depends); err == nil {
		if env != currentEnv {
			if currentEnv != "" {
				if err := gocdChangeEnv(resource, currentEnv, map[string]interface{}{"remove": []string{name}}); err != nil {
					return err
				}
			}

			if err := gocdChangeEnv(resource, env, map[string]interface{}{"add": []string{name}}); err != nil {
				return err
			}
		}
	} else {
		return err
	}

	return nil
}

func goCdSchedule(name string, resource string, body string, headers map[string]string) error {
	_, err := goCdRequest("POST", resource+"/go/api/pipelines/"+name+"/schedule", body, headers)
	return err
}

func goCdDelete(name string, env string, resource string, headers map[string]string) error {
	if err := gocdChangeEnv(resource, env, map[string]interface{}{"remove": []string{name}}); err != nil {
		log.Println("Error on remove pipeline from env: ", err)
	}

	if resp, err := goCdRequest("DELETE", resource+"/go/api/admin/pipelines/"+name, "", headers); err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Operation error: %s", resp.Status)
	}

	return nil
}

func gocdChangeEnv(apiUrl string, env string, actions map[string]interface{}) error {
	body, _ := json.Marshal(map[string]interface{}{
		"pipelines": actions,
	})

	resp, err := goCdRequest("PATCH", apiUrl+"/go/api/admin/environments/"+env, string(body),
		map[string]string{"Accept": "application/vnd.go.cd.v3+json"})
	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Operation error: %s", resp.Status)
	}

	return nil
}

func goCdFindEnv(resource string, pipeline string, depends []string) (string, error) {
	resp, err := goCdRequest("GET", resource+"/go/api/admin/environments", "",
		map[string]string{"Accept": "application/vnd.go.cd.v3+json"})
	if err != nil {
		return "", err
	} else if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Operation error: %s", resp.Status)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	tree, err := gabs.ParseJSON(body)
	if err != nil {
		return "", err
	}

	sort.Strings(depends)
	envs, _ := tree.Path("_embedded.environments").Children()
	curEnvName := ""
	for _, env := range envs {
		envName := env.Path("name").Data().(string)
		pipelines, _ := env.Path("pipelines").Children()

		if len(depends) > 0 {
			if i := sort.SearchStrings(depends, envName); i != len(depends) {
				depends = append(depends[:i], depends[i+1:]...)
			}
		} else {
			if curEnvName != "" {
				break
			}
		}

		for _, pline := range pipelines {
			if pline.Path("name").Data().(string) == pipeline {
				curEnvName = envName
			}
		}
	}

	if len(depends) != 0 {
		return curEnvName, fmt.Errorf("not found depends: %v", depends)
	}

	return curEnvName, nil
}

var goCdRequest = func(method string, url string, body string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader([]byte(body)))
	if err != nil {
		log.Printf(" --> %s %s — %v", method, url, err)
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	req.Header.Set("Content-Type", "application/json")

	data, err := ioutil.ReadFile("/etc/serve/gocd_credentials")
	if err != nil {
		log.Printf(" --> %s %s — %v", method, url, err)
		return nil, fmt.Errorf("Credentias file error: %v", err)
	}

	creds := &goCdCredents{}
	json.Unmarshal(data, creds)

	req.SetBasicAuth(creds.Login, creds.Password)

	log.Printf(" --> %s %s:\n%s\n\n", method, url, body)

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf(" --> %s %s — %v", method, url, err)
		return nil, err
	}

	log.Printf("<-- %s", resp.Status)
	return resp, nil
}
