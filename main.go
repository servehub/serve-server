package main

import (
	"fmt"
	golog "log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/cenkalti/backoff"
	"github.com/kulikov/go-sbus"
	"github.com/kulikov/go-sbus/transports"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/servehub/utils/gabs"

	"github.com/servehub/serve-server/config"
	"github.com/servehub/serve-server/handler"
	_ "github.com/servehub/serve-server/handlers"
)

var version = "1.0"

func main() {
	configPath := kingpin.Flag("config", "Path to config.yml file.").Default("config.yml").String()

	kingpin.Version(version)
	kingpin.Parse()

	conf, err := gabs.LoadYamlFile(*configPath)
	kingpin.FatalIfError(err, "Error on load config file: %s", configPath)

	err = conf.WithFallbackYaml(config.MustAsset("config/reference.yml"))
	kingpin.FatalIfError(err, "Error on load reference config")

	level, err := logrus.ParseLevel(fmt.Sprintf("%v", conf.Path("logging.level").Data()))
	kingpin.FatalIfError(err, "Unknown logging level: %v", conf.Path("logging.level").Data())

	logrus.SetLevel(level)

	switch fmt.Sprintf("%v", conf.Path("logging.formatter").Data()) {
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{
			DisableTimestamp: true,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyMsg: "message",
			},
		})

	default:
		logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	}

	log := logrus.NewEntry(logrus.StandardLogger())

	golog.SetFlags(0)
	golog.SetOutput(log.Logger.Writer())

	bus := sbus.New(transports.NewInMemory(log), log)

	log.WithField("version", version).Infoln("Start serve-server")

	handlersConf, _ := conf.Path("handlers").Children()

	for _, handlerItem := range handlersConf {
		handlerMap, _ := handlerItem.ChildrenMap()

		for handlerName, handlerConf := range handlerMap {
			if !handler.HandlerRegestry.Has(handlerName) {
				log.Errorf("Handler %s doesn't exists!", handlerName)
				continue
			}

			hndr := handler.HandlerRegestry.Get(handlerName)

			go func(hndr handler.Handler, handlerConf *gabs.Container, log *logrus.Entry, handlerName string) {
				log.WithField("config", handlerConf.Data()).Infof("Starting `%s`...", handlerName)

				err := backoff.RetryNotify(func() error {
					return hndr.Run(bus, handlerConf, log)
				}, backoff.NewExponentialBackOff(), func(err error, delay time.Duration) {
					log.WithError(err).Errorf("Error in handler `%s`. Retry after %s...", handlerName, delay)
				})

				if err != nil {
					log.WithError(err).Errorln("Error on initialize handler", handlerName)
				}
			}(hndr, handlerConf, log.WithField("handler", handlerName), handlerName)
		}
	}

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	log.Infoln("serve-server shutdown by", <-ch)
}
