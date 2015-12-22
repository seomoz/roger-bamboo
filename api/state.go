package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/samuel/go-zookeeper/zk"

	"github.com/seomoz/roger-bamboo/configuration"
	"github.com/seomoz/roger-bamboo/services/haproxy"
)

type StateAPI struct {
	Config    *configuration.Configuration
	Zookeeper *zk.Conn
}

func (state *StateAPI) Get(w http.ResponseWriter, r *http.Request) {
	payload, _ := json.Marshal(haproxy.GetTemplateData(state.Config, state.Zookeeper))
	io.WriteString(w, string(payload))
}
