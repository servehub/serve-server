package webhooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/kulikov/go-sbus"

	"github.com/servehub/utils/gabs"

	"github.com/servehub/serve-server/handler"
)

func init() {
	handler.HandlerRegestry.Add("webhooks-pingdom", &WebhooksPingdom{})
}

type WebhooksPingdom struct{}

func (_ WebhooksPingdom) Run(bus *sbus.Sbus, conf *gabs.Container, log *logrus.Entry) error {

	bus.Sub("receive-webhook-pingdom", func(msg sbus.Message) error {
		data, err := gabs.ParseJSON(msg.Data)
		if err != nil {
			return err
		}

		log.Debugln("Receive pingdom webhook: ", data.StringIndent("", "  "))

		color := "#CCC"
		if data.Path("current_state").Data() == "DOWN" {
			color = "#F00"
		} else if data.Path("current_state").Data() == "UP" {
			color = "#2EB886"
		}

		body, _ := json.Marshal(map[string]interface{}{
			"username": "Pingdom",
			"text": fmt.Sprintf("%s \n\n*%s* \n\n%s", data.Path("check_params.full_url").Data(), data.Path("current_state").Data(), data.Path("long_description").Data()),
			"attachments": []interface{}{
				map[string]string{
	        "color": color,
	        "text":  "```" + data.StringIndent("", "  ") + "```",
				},
      },
		})

		req, _ := http.NewRequest("POST", conf.Path("slack-url").Data().(string), bytes.NewBuffer(body))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode != 200 {
			return fmt.Errorf("Slack error response: %v", resp.Status)
		}

		return nil
	})

	return nil
}
