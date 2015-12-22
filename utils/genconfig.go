package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/samuel/go-zookeeper/zk"
	"github.com/seomoz/roger-bamboo/configuration"
	"github.com/seomoz/roger-bamboo/services/haproxy"
	"github.com/seomoz/roger-bamboo/services/template"
	lumberjack "github.com/natefinch/lumberjack"
)


/* Commandline arguments */
var configFilePath string
var logPath string

func init() {
	flag.StringVar(&configFilePath, "config", "config/development.json", "Full path of the configuration JSON file")
	flag.StringVar(&logPath, "log", "", "Log path to a file. Default logs to stdout")
}

func main() {
	flag.Parse()
	configureLog()
	log.Println("Using config from " + configFilePath)

	// Load configuration
	conf, err := configuration.FromFile(configFilePath)
	if err != nil {
		log.Fatal(err)
	}

	// Load the Haproxy template file.
	log.Println("Loading HAProxy template from " + conf.HAProxy.TemplatePath)
	templateContent, err := ioutil.ReadFile(conf.HAProxy.TemplatePath)
	if err != nil { log.Panicf("Cannot read template file: %s", err) }

	zkConf := conf.Bamboo.Zookeeper
	//log.Println("Connecting to Zookeeper using " + zkConf.ConnectionString())
	conn, _, err := zk.Connect(zkConf.ConnectionString(), time.Second*10)
	if err != nil {
		log.Panic(err)
	}

	// Get the App config data from Marathon.
	templateData := haproxy.GetTemplateData(&conf, conn)

	// Render the template
	newContent, err := template.RenderTemplate(conf.HAProxy.TemplatePath, string(templateContent), templateData)
	if err != nil { log.Fatalf("Template syntax error: \n %s", err ) }

	// Write the rendered template.
	log.Println("===============Begin HAProxy config===========================");
	log.Println(newContent);
	log.Println("===============End HAProxy config===========================");
	
}

func configureLog() {
	if len(logPath) > 0 {
		log.SetOutput(io.MultiWriter(&lumberjack.Logger{
			Filename: logPath,
			// megabytes
			MaxSize:    100,
			MaxBackups: 3,
			//days
			MaxAge: 28,
		}, os.Stdout))
	}
}
