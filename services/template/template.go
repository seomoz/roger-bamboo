package template

import (
	"bytes"
	"fmt"
	"github.com/seomoz/roger-bamboo/services/service"
	"hash/fnv"
	"regexp"
	"strconv"
	"text/template"
	"time"
	"strings"
	"sort"
)

// The set of regexes defining valid port descriptions. a valid
// description is either a number or of the form PORTX where x is a
// number. See the getTaskPort method.
var taskPortRegex = regexp.MustCompile("^PORT([\\d]+)$")
var numPortRegex = regexp.MustCompile("^[\\d]+$")
var hasher = fnv.New32a()
var hasher64 = fnv.New64a()

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
		match := taskPortRegex.FindStringSubmatch(port_description)
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

/* Given a server's ID, host and port, returns the combined hash of
the inputs. This function is called to compute a unique hash for a
given server instance which can then be set on a cookie to enable
session affinity.*/
func getServerHash(escapeId string, host string, port int) string {
	hasher.Write([]byte(escapeId))
	hasher.Write([]byte(host))
	portString := []byte(strconv.Itoa(port))
	hasher.Write(portString)
	hash := hasher.Sum32()
	hasher.Reset()
	return fmt.Sprintf("%X", hash)
}

/* Compute a hash for the given string */
func getHash(str string) string {
	hasher64.Write([]byte(str))
	hash := hasher64.Sum64()
	hasher64.Reset()
	return fmt.Sprintf("%X", hash)
}

/* Replaces "/" with "::" */
func escapeSlashes(someString string) string {
	return strings.Replace(someString, "/", "::", -1)
}

/* Adds the acl into the acls set */
func addAcl(acls map[string]bool, acl string) string {
	acls[acl] = true
	return ""
}

/* Adds (updates) the backed rule (as value) for the given condition (as key) into the backend rule map */
func addBackendRule(backendrules map[string]string, backend string, condition string) string {
	backendrules[condition] = backend
	return ""
}

/* Returns the conditions (keys) in descending order from backendrules map */
func getConditionsDescending(backendrules map[string]string) []string {
	conditions := make([]string, len(backendrules))
	i := 0
	for k := range backendrules {
		conditions[i] = k
		i++
	}
	sort.Sort(sort.Reverse(sort.StringSlice(conditions)))
	return conditions
}

/*
	Returns string content of a rendered template
*/
func RenderTemplate(templateName string, templateContent string, data interface{}) (string, error) {
	funcMap := template.FuncMap{"hasKey": hasKey, "getService": getService, "getTime": getTime, "getTaskPort": getTaskPort, "getServerHash": getServerHash, "getHash": getHash, "escapeSlashes": escapeSlashes, "addAcl": addAcl, "addBackendRule": addBackendRule, "getConditionsDescending": getConditionsDescending }

	tpl := template.Must(template.New(templateName).Funcs(funcMap).Parse(templateContent))

	strBuffer := new(bytes.Buffer)

	err := tpl.Execute(strBuffer, data)
	if err != nil {
		return "", err
	}

	return strBuffer.String(), nil
}
