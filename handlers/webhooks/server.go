package webhooks

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/Sirupsen/logrus"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/kulikov/go-sbus"

	"github.com/servehub/serve-server/handler"
	"github.com/servehub/utils/gabs"
)

func init() {
	handler.HandlerRegestry.Add("webhooks-server", &WebhooksServer{})
}

var webhookName = regexp.MustCompile("[^a-z\\d]+")

type WebhooksServer struct{}

func (_ WebhooksServer) Run(bus *sbus.Sbus, conf *gabs.Container, log *logrus.Entry) error {

	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestLogger(&StructuredLogger{log}))
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	secretKey := fmt.Sprintf("%v", conf.Path("secret-key").Data())

	r.Post("/*", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Query().Get("key") != secretKey {
			log.Errorf("Incorrect secret key in url " + req.URL.Query().Get("key") + "!")
			http.Error(w, http.StatusText(403), 403)
			return
		}

		body, _ := ioutil.ReadAll(req.Body)
		defer req.Body.Close()

		_ = bus.PubM(sbus.Message{
			Subject: "receive-webhook-" + webhookName.ReplaceAllString(req.URL.Path[1:], "-"),
			Data:    body,
			Meta: sbus.Meta{
				"body":    body,
				"headers": req.Header,
			},
		})

		w.WriteHeader(http.StatusAccepted)
	})

	return http.ListenAndServe(fmt.Sprintf("%v", conf.Path("listen").Data()), r)
}
