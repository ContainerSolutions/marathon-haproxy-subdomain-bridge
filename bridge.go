package main

import (
    "flag"
    "fmt"
    "os"
    "net/http"
    "time"
    "bufio"
    "strings"
    "io/ioutil"
    "bytes"
)

const (

    domain = "laurasjourney.nl"

    defaultBackend = "journey"

    help_text = `
     USAGE: $name <marathon host:port>+
        $name install_haproxy_system <marathon host:port>+

      Generates a new configuration file for HAProxy from the specified Marathon
      servers, replaces the file in /etc/haproxy and restarts the service.

      In the second form, installs the script itself, HAProxy and a cronjob that
      once a minute pings one of the Marathon servers specified and refreshes
      HAProxy if anything has changed. The list of Marathons to ping is stored,
      one per line, in:

      $cronjob_conf_file

      The script is installed as:

      $script_path

      The cronjob is installed as:

      $cronjob

      and run as root.
    `

    header = `global
  daemon
  log 127.0.0.1 local0
  log 127.0.0.1 local1 notice
  log /var/lib/haproxy/dev/log local0
  maxconn 4096
  tune.ssl.default-dh-param 2048

defaults
  mode           http
  log            global
  retries             3
  maxconn          2000
  timeout connect  5000
  timeout client  50000
  timeout server  50000

listen stats
  bind 127.0.0.1:9090
  balance
  mode http
  stats enable
  stats auth admin:admin
`

    frontendStart = `frontend http-in
  bind :80
  bind :443 ssl crt /etc/haproxy/site.pem`

    aclFormat = `  acl subdomain-%s hdr_dom(host) -i %s.%s
  use_backend %s if subdomain-%s`

    frontendEnd = `  default_backend %s`

    listen = `listen %s-%s
  bind 0.0.0.0:%s
  mode tcp
  option tcplog
  balance leastconn`

    server = `  server %s-%d %s check`

    backend = `backend %s
  balance leastconn
  option httpclose
  option forwardfor`

)

type TasksFetcher interface {
    FetchTasks(hostport string) ([]byte, error)
}

type MarathonTaskFetcher struct {

}

func (f MarathonTaskFetcher) FetchTasks(hostport string) ([]byte, error) {
    client := http.Client{}
    client.Timeout = time.Duration(60 * time.Second)

    req, err := http.NewRequest("GET", "http://" + hostport + "/v2/tasks", nil)
    checkerr(err)

    req.Header.Add("Accept",  "text/plain")
    resp, err := client.Do(req)
    checkerr(err)

    defer resp.Body.Close()

    contents, err := ioutil.ReadAll(resp.Body)

    if err != nil {
        return nil, err
    }

    return contents, nil
}

type Acl struct {
    App string
    Port string
}

func main() {
    help := flag.Bool("help", false, help_text)
    flag.Parse()

    var hostport string
    if *help || len(os.Args) <= 1 {
        fmt.Println(help_text)
        os.Exit(0)
    } else {
        hostport = os.Args[1]
    }

    fetcher := MarathonTaskFetcher{}

    config := generateHaProxyConfig(fetcher, hostport)

    fmt.Println(config)
}

func generateHaProxyConfig(fetcher TasksFetcher, hostport string) string {
    var config = header

    contents, err := fetcher.FetchTasks(hostport)
    checkerr(err)

    scanner := bufio.NewScanner(bytes.NewReader(contents))
    var acls []Acl
    for scanner.Scan() {
        fields := strings.Split(scanner.Text(), "\t")

        acl := Acl{App: fields[0], Port: fields[1]}
        acls = append(acls, acl)

        if acl.Port == "80" {
            config = config + generateBackend(fields[2:], acl)
        } else {
            config = config +  generateListen(fields[2:], acl)
        }

        config = config + "\n"
    }

    return config + generateFrontend(acls)
}

func generateFrontend(acls []Acl) string {
    var config string

    config = fmt.Sprintf(frontendStart + "\n")

    for _, acl := range acls {
        if acl.Port == "80" {
            strippedApp := strings.Replace(acl.App, "lauras-", "", -1)
            config = config + fmt.Sprintf(aclFormat + "\n", acl.App, strippedApp, domain, strippedApp, acl.App)
        }
    }

    return config + fmt.Sprintf(frontendEnd + "\n", defaultBackend)
}

func generateListen(servers []string, acl Acl) string {
    var config string

    config = fmt.Sprintf(listen + "\n", acl.App, acl.Port, acl.Port)

	return config + generateServers(servers, acl)
}

func generateBackend(servers []string, acl Acl) string {
    strippedapp := strings.Replace(acl.App, "lauras-", "", -1)

    config := fmt.Sprintf(backend + "\n", strippedapp)

	return config + generateServers(servers, acl)
}

func generateServers(servers []string, acl Acl) string {
    var config string
	for index, servername := range servers {
		if servername != "" {
			config = config + fmt.Sprintf(server + "\n", acl.App, index + 1, servername)
		}
	}
    return config
}

func checkerr(err error) {
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
