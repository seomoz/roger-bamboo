package configuration

import (
	"strings"
)

/*
	Mesos Marathon configuration
*/
type Marathon struct {
	// comma separated marathon http endpoints including port number
	Endpoint string

	// this controls how often bamboo polls marathon for new applications
	// (unit is seconds, minimum is 1 second, default is 30 seconds)
	PollingInterval int
}

func (m Marathon) Endpoints() []string {
	return strings.Split(m.Endpoint, ",")
}
