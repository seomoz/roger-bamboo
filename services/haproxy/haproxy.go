package haproxy

import (
	"github.com/samuel/go-zookeeper/zk"

	conf "github.com/seomoz/roger-bamboo/configuration"
	"github.com/seomoz/roger-bamboo/services/marathon"
	"github.com/seomoz/roger-bamboo/services/service"
)

type TemplateData struct {
	Apps     marathon.AppList
	Services map[string]service.Service
	Acls map[string]bool
	BackendRules map[string]string
}

func GetTemplateData(config *conf.Configuration, conn *zk.Conn) TemplateData {

	apps, _ := marathon.FetchApps(config.Marathon)
	services, _ := service.All(conn, config.Bamboo.Zookeeper)
	acls := make(map[string]bool)
	backendrules := make(map[string]string)

	return TemplateData{apps, services, acls, backendrules}
}
