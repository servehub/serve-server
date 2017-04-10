package handlers

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/kulikov/go-sbus"
	"github.com/pressly/chi"
	"github.com/pressly/chi/middleware"

	"github.com/servehub/serve-server/handler"
	"github.com/servehub/utils/gabs"
)

func init() {
	handler.HandlerRegestry.Add("webhooks-server", &WebhooksServer{})
}

type WebhooksServer struct{}

func (_ WebhooksServer) Run(bus *sbus.Sbus, conf *gabs.Container, log *logrus.Entry) error {

	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestLogger(&StructuredLogger{log}))
	r.Use(middleware.Recoverer)

	r.HandleFunc("/*", func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, req.Method+" "+req.RequestURI+"\n\n")

		for k, v := range req.Header {
			io.WriteString(w, k+": "+strings.Join(v, "; ")+"\n")
		}

		body, _ := ioutil.ReadAll(req.Body)
		defer req.Body.Close()

		io.WriteString(w, "\n")
		w.Write(body)

		io.WriteString(w, "\n\nHello, "+os.Getenv("SERVICE_NAME")+"!\n\n\n\n")
	})

	return http.ListenAndServe(fmt.Sprintf("%v", conf.Path("listen").Data()), r)
}
