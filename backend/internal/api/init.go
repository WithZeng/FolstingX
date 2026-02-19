package api

import (
	"github.com/folstingx/server/config"
	"github.com/folstingx/server/internal/services"
)

type appContext struct {
	cfg       *config.Config
	forwarder *services.ForwardManager
	collector *services.TrafficCollector
	xray      *services.XrayManager
	gost      *services.GostManager
	hub       *MonitorHub
}

var app *appContext

func Init(cfg *config.Config, fm *services.ForwardManager, tc *services.TrafficCollector, xray *services.XrayManager, gost *services.GostManager) {
	app = &appContext{
		cfg:       cfg,
		forwarder: fm,
		collector: tc,
		xray:      xray,
		gost:      gost,
		hub:       NewMonitorHub(),
	}
	app.hub.Start()
}
