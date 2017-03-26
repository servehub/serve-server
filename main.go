package main

import (
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

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log := logrus.NewEntry(logrus.StandardLogger())

	bus := sbus.New(transports.NewInMemory(log), log)

	log.Infoln("Start serve-server")

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
				log.WithField("config", handlerConf).Infof("Starting `%s`...", handlerName)

				err := backoff.RetryNotify(func() error {
					return hndr.Run(bus, handlerConf, log)
				}, backoff.NewExponentialBackOff(), func(err error, delay time.Duration) {
					log.WithError(err).Errorf("Error on initialize handler `%s`. Retry after %s...", handlerName, delay)
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
