package handlers

import (
	_ "github.com/servehub/serve-server/handlers/consul"
	_ "github.com/servehub/serve-server/handlers/webhooks"
	_ "github.com/servehub/serve-server/handlers/gocd"
	_ "github.com/servehub/serve-server/handlers/serve"
	_ "github.com/servehub/serve-server/handlers/dashboards"
)
