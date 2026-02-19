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
	agentHub  *services.AgentHub
}

var app *appContext

func Init(cfg *config.Config, fm *services.ForwardManager, tc *services.TrafficCollector, xray *services.XrayManager, gost *services.GostManager, ah *services.AgentHub) {
	app = &appContext{
		cfg:       cfg,
		forwarder: fm,
		collector: tc,
		xray:      xray,
		gost:      gost,
		hub:       NewMonitorHub(),
		agentHub:  ah,
	}
	app.hub.Start()
}
