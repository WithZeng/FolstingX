package forwarder

import "time"

type Stats struct {
  UpBytes      int64     `json:"up_bytes"`
  DownBytes    int64     `json:"down_bytes"`
  Connections  int64     `json:"connections"`
  LastActivity time.Time `json:"last_activity"`
}

type Forwarder interface {
  Start() error
  Stop() error
  Stats() Stats
}
