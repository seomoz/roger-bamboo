package event_bus

import (
	"bytes"
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/seomoz/roger-bamboo/configuration"
	"github.com/seomoz/roger-bamboo/services/haproxy"
	"github.com/seomoz/roger-bamboo/services/template"
	"hash/fnv"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"reflect"
	"strings"
)

type MarathonEvent struct {
	// EventType can be
	// api_post_event, status_update_event, subscribe_event
	EventType string
	Timestamp string
}

type ZookeeperEvent struct {
	Source    string
	EventType string
}

type ServiceEvent struct {
	EventType string
}

type Handlers struct {
	Conf      *configuration.Configuration
	Zookeeper *zk.Conn
}

func (h *Handlers) MarathonEventHandler(event MarathonEvent) {
	log.Printf("%s => %s\n", event.EventType, event.Timestamp)
	queueUpdate(h)
	h.Conf.StatsD.Increment(1.0, "reload.marathon", 1)
}

func (h *Handlers) ServiceEventHandler(event ServiceEvent) {
	log.Println("Domain mapping: Stated changed")
	queueUpdate(h)
	h.Conf.StatsD.Increment(1.0, "reload.domain", 1)
}

var updateChan = make(chan *Handlers, 1)

var currentConfig = ""
var currentConfigHash = ""
var hasher = fnv.New64a()                    // The hash function
var currentTemplateData haproxy.TemplateData // The current data from Marathon

/* Called by the webserver to report the hash of the current config. */
func GetCurrentConfigHash(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, currentConfigHash)
}

/* Called by the webserver to report the current config file. */
func GetCurrentConfig(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, currentConfig)
}

/* Called by the webserver to report the list of currently used ports. */
func GetUsedPorts(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	for _, app := range currentTemplateData.Apps {
		for port, _ := range app.TcpPorts {
			buf.WriteString(fmt.Sprintf("%s : %s \n", port, app.Id))
		}
	}
	io.WriteString(w, buf.String())
}

func init() {
	go func() {
		log.Println("Starting update loop ...")
		for {
			h := <-updateChan
			log.Println("Got request for new update")
			handleHAPUpdate(h.Conf, h.Zookeeper)
			log.Println("Finished processing new update")
		}
	}()
}

var queueUpdateSem = make(chan int, 1)

func queueUpdate(h *Handlers) {
	queueUpdateSem <- 1
	select {
	case _ = <-updateChan:
		log.Println("Found pending update request. Don't start another one.")
	default:
		log.Println("Queuing an haproxy update.")
	}
	updateChan <- h
	<-queueUpdateSem
}

func handleHAPUpdate(conf *configuration.Configuration, conn *zk.Conn) bool {
	templateContent, err := ioutil.ReadFile(conf.HAProxy.TemplatePath)
	if err != nil {
		log.Panicf("Cannot read template file: %s", err)
	}

	// The first line in the template just contains the time the
	// template was rendered. The rest of the template is
	// idempotent and only relies on the data we get from
	// marathon. To report the hash of the rendered config, we
	// create a second template which omits the first line (and
	// hence the part which can differ across machines). The
	// second template is used to compute the hash.
	idempotentTemplate := strings.Replace(string(templateContent), "# Template rendered at {{ getTime }}", "", 1)

	templateData := haproxy.GetTemplateData(conf, conn)

	newContent, err := template.RenderTemplate(conf.HAProxy.TemplatePath, string(templateContent), templateData)
	if err != nil {
		log.Fatalf("Template syntax error: \n %s", err)
	}

	newIdempotentContent, err := template.RenderTemplate("IdempotentTemplate",
		string(idempotentTemplate), templateData)
	if err != nil {
		log.Fatalf("Idempotent Template syntax error: \n %s", err)
	}

	if !reflect.DeepEqual(currentTemplateData, templateData) {
		err := ioutil.WriteFile(conf.HAProxy.OutputPath, []byte(newContent), 0666)
		if err != nil {
			log.Fatalf("Failed to write template on path: %s", err)
			return false
		}
		// Now that the config file corresponding to the new
		// template data has been written, update the
		// currentTemplateData variable with the new data.
		currentTemplateData = templateData
		err = execCommand(conf.HAProxy.ReloadCommand)
		if err != nil {
			log.Fatalf("HAProxy: update failed\n")
		} else {
			log.Println("HAProxy: Configuration updated")
			// Now that the HAproxy config has been
			// updated, start exporting the new values.
			currentConfig = newIdempotentContent
			hasher.Write([]byte(currentConfig))
			currentConfigHash = fmt.Sprintf("%X", hasher.Sum64())
		}
		return true
	} else {
		log.Println("HAProxy: Same content, no need to reload")
		return false
	}
}

func execCommand(cmd string) error {
	log.Printf("Exec cmd: %s \n", cmd)
	output, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		log.Println(err.Error())
		log.Println("Problem executing command Output:\n" + string(output[:]))
	}
	log.Println("Finished running command")
	return err
}
