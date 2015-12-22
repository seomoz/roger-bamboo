package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/kardianos/osext"
	lumberjack "github.com/natefinch/lumberjack"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/zenazn/goji"

	"github.com/seomoz/roger-bamboo/api"
	"github.com/seomoz/roger-bamboo/configuration"
	"github.com/seomoz/roger-bamboo/qzk"
	"github.com/seomoz/roger-bamboo/services/event_bus"
)

/*
	Commandline arguments
*/
var configFilePath string
var logPath string

func init() {
	flag.StringVar(&configFilePath, "config", "config/development.json", "Full path of the configuration JSON file")
	flag.StringVar(&logPath, "log", "", "Log path to a file. Default logs to stdout")
}

func main() {
	log.Println("Starting binary..")
	flag.Parse()
	configureLog()

	// Load configuration
	conf, err := configuration.FromFile(configFilePath)
	if err != nil {
		log.Fatal(err)
	}

	eventBus := event_bus.New()

	// Wait for died children to avoid zombies
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGCHLD)
	go func() {
		for {
			sig := <-signalChannel
			if sig == syscall.SIGCHLD {
				r := syscall.Rusage{}
				syscall.Wait4(-1, nil, 0, &r)
			}
		}
	}()

	log.Println("Created statsD client")
	// Create StatsD client
	conf.StatsD.CreateClient()

	// Create Zookeeper connection
	zkConn := listenToZookeeper(conf, eventBus)

	// Register handlers
	handlers := event_bus.Handlers{Conf: &conf, Zookeeper: zkConn}
	eventBus.Register(handlers.MarathonEventHandler)
	eventBus.Register(handlers.ServiceEventHandler)
	log.Println("Registered handlers")

	// Start periodic config updates
	ticker := time.Tick(30 * time.Second)
	go func() {
		for {
			<-ticker
			// Simulate a service event to write out the HAproxy config initially
			log.Println("Simulating service event....")
			eventBus.Publish(event_bus.ServiceEvent{EventType: "change"})
		}
	}()

	// Start server
	initServer(&conf, zkConn, eventBus)
}

func initServer(conf *configuration.Configuration, conn *zk.Conn, eventBus *event_bus.EventBus) {
	log.Println("in initServer")
	stateAPI := api.StateAPI{Config: conf, Zookeeper: conn}
	serviceAPI := api.ServiceAPI{Config: conf, Zookeeper: conn}
	eventSubAPI := api.EventSubscriptionAPI{Conf: conf, EventBus: eventBus}

	log.Println("in initServer 2")

	conf.StatsD.Increment(1.0, "restart", 1)
	// Status live information
	goji.Get("/status", api.HandleStatus)

	// Current config and it's hash
	goji.Get("/config", event_bus.GetCurrentConfig)
	goji.Get("/confighash", event_bus.GetCurrentConfigHash)

	// Currently used ports
	goji.Get("/usedports", event_bus.GetUsedPorts)

	// State API
	goji.Get("/api/state", stateAPI.Get)

	// Service API
	goji.Get("/api/services", serviceAPI.All)
	goji.Post("/api/services", serviceAPI.Create)
	goji.Put("/api/services/:id", serviceAPI.Put)
	goji.Delete("/api/services/:id", serviceAPI.Delete)
	goji.Post("/api/marathon/event_callback", eventSubAPI.Callback)

	log.Println("in initServer 3")
	// Static pages
	goji.Get("/*", http.FileServer(http.Dir(path.Join(executableFolder(), "webapp"))))

	log.Println("in initServer 4")
	registerMarathonEvent(conf)

	goji.Serve()
}

// Get current executable folder path
func executableFolder() string {
	folderPath, err := osext.ExecutableFolder()
	if err != nil {
		log.Fatal(err)
	}
	return folderPath
}

func registerMarathonEvent(conf *configuration.Configuration) {
	log.Println("in registerMarathonEvent 1")
	client := &http.Client{}
	// it's safe to register with multiple marathon nodes
	for _, marathon := range conf.Marathon.Endpoints() {
		url := marathon + "/v2/eventSubscriptions?callbackUrl=" + conf.Bamboo.Endpoint + "/api/marathon/event_callback"
		req, _ := http.NewRequest("POST", url, nil)
		req.Header.Add("Content-Type", "application/json")
		log.Println("calling " + url)
		client.Do(req)
	}
	log.Println("in registerMarathonEvent 2")
}

func createAndListen(conf configuration.Zookeeper) (chan zk.Event, *zk.Conn) {
	conn, _, err := zk.Connect(conf.ConnectionString(), time.Second*10)

	if err != nil {
		log.Panic(err)
	}

	ch, _ := qzk.ListenToConn(conn, conf.Path, true, conf.Delay())
	return ch, conn
}

func listenToZookeeper(conf configuration.Configuration, eventBus *event_bus.EventBus) *zk.Conn {
	serviceCh, serviceConn := createAndListen(conf.Bamboo.Zookeeper)

	go func() {
		for {
			select {
			case _ = <-serviceCh:
				eventBus.Publish(event_bus.ServiceEvent{EventType: "change"})
			}
		}
	}()
	return serviceConn
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
