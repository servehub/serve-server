package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"github.com/Sirupsen/logrus"
	"github.com/kulikov/go-sbus"
	"github.com/pressly/chi"
	"github.com/pressly/chi/middleware"
	"github.com/pressly/chi/render"

	"github.com/servehub/serve-server/handler"
	"github.com/servehub/utils/gabs"
)

func init() {
	handler.HandlerRegestry.Add("debug-handler", &DebugHandler{})
}

type DebugHandler struct{}

func (_ DebugHandler) Run(bus *sbus.Sbus, conf *gabs.Container, log *logrus.Entry) error {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Post("/sbus/pub/:subject", func(w http.ResponseWriter, req *http.Request) {
		body, _ := ioutil.ReadAll(req.Body)
		defer req.Body.Close()

		data, err := gabs.ParseJSON(body)
		if err != nil {
			http.Error(w, http.StatusText(400), 400)
			return
		}

		bus.Pub(chi.URLParam(req, "subject"), data.Data())

		w.WriteHeader(http.StatusAccepted)
	})

	return http.ListenAndServe(fmt.Sprintf("%v", conf.Path("listen").Data()), r)
}
