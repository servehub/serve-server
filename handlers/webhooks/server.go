package webhooks

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/Sirupsen/logrus"
	"github.com/kulikov/go-sbus"
	"github.com/pressly/chi"
	"github.com/pressly/chi/middleware"
	"github.com/pressly/chi/render"

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
			http.Error(w, http.StatusText(403), 403)
			return
		}

		body, _ := ioutil.ReadAll(req.Body)
		defer req.Body.Close()

		data, err := gabs.ParseJSON(body)
		if err != nil {
			http.Error(w, http.StatusText(400), 400)
			return
		}

		bus.Pub("receive-webhook-"+webhookName.ReplaceAllString(req.URL.Path[1:], "-"), data.Data())

		w.WriteHeader(http.StatusAccepted)
	})

	return http.ListenAndServe(fmt.Sprintf("%v", conf.Path("listen").Data()), r)
}
