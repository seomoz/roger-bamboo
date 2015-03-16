package template

import (
	"bytes"
	"github.com/seomoz/roger-bamboo/services/service"
	"regexp"
	"strconv"
	"fmt"
	"text/template"
	"time"
)

// The set of regexes defining valid port descriptions. a valid
// description is either a number or of the form PORTX where x is a
// number. See the getTaskPort method.
var taskPortRegex = regexp.MustCompile("^PORT([\\d]+)$")
var numPortRegex = regexp.MustCompile("^[\\d]+$")

func hasKey(data map[string]service.Service, appId string) bool {
	_, exists := data[appId]
	return exists
}

func getService(data map[string]service.Service, appId string) service.Service {
	serviceModel, _ := data[appId]
	return serviceModel
}

/* Returns the current time as a string */
func getTime() string {
	return time.Now().String()
}

/* Given the array of local ports on a task and a string describing
  the port to be picked. Returns the appropriate port. If
  port_description is a number then its returned as is. If
  port_description is for the form PORTX where X is a number, returns
  the value Ports[X]. If port_description has any other format, a
  panic is raise. The lack of error checking on the indexing is
  deliberate and intended to cause the template rendering to fail
  rather than have an incorrect value.  E.g given Ports = [21334,
  312333] and port_description = PORT0 will return 21334*/
func getTaskPort(Ports []int, port_description string) string {
	// If the port desctiption is of the form PORTX
	if taskPortRegex.MatchString(port_description) {
		match:= taskPortRegex.FindStringSubmatch(port_description)
		portIndex, err := strconv.Atoi(match[1])
		if err != nil {
			panic(fmt.Sprintf("Unable to convert %s to int while processing port decription %s", match[1], port_description))
		}
		return strconv.Itoa(Ports[portIndex])
	}
	// If the port description is a port number.
	if numPortRegex.MatchString(port_description) {
		// Return the port number itself.
		return port_description
	}

	// The port_description is not valid
	panic(fmt.Sprintf("Invalid port_description %s", port_description))
}

/*
	Returns string content of a rendered template
*/
func RenderTemplate(templateName string, templateContent string, data interface{}) (string, error) {
	funcMap := template.FuncMap{ "hasKey": hasKey,  "getService": getService, "getTime": getTime, "getTaskPort": getTaskPort }

	tpl := template.Must(template.New(templateName).Funcs(funcMap).Parse(templateContent))

	strBuffer := new(bytes.Buffer)

	err := tpl.Execute(strBuffer, data)
	if err != nil {
		return "", err
	}

	return strBuffer.String(), nil
}

