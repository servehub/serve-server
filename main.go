package main

import (
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/servehub/serve-server/config"

	"github.com/servehub/utils/gabs"
)

var version = "0.5.0"

func main() {
	configPath := kingpin.Flag("config", "Path to config.yml file.").Default("config.yml").String()

	kingpin.Version(version)
	kingpin.Parse()

	conf, err := gabs.LoadYamlFile(*configPath)
	kingpin.FatalIfError(err, "Error on load config file: %s", configPath)

	err = conf.WithFallbackYaml(config.MustAsset("config/reference.yml"))
	kingpin.FatalIfError(err, "Error on load reference config")

	println(conf.StringIndent("", "  "))
}
