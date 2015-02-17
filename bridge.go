package main

import (
    "flag"
    "fmt"
    "os"
    "net/http"
    "time"
    "bufio"
    "strings"
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
  maxconn 4096

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

    aclFormat = `  acl subdomain-%s hdr(host) -i %s.%s
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

    client := http.Client{}
    client.Timeout = time.Duration(60 * time.Second)

    req, err := http.NewRequest("GET", "http://" + hostport + "/v2/tasks", nil)
    checkerr(err)

    req.Header.Add("Accept",  "text/plain")
    resp, err := client.Do(req)
    checkerr(err)

    defer resp.Body.Close()

    fmt.Println(header)

    scanner := bufio.NewScanner(resp.Body)
    // Skip first empty line
    scanner.Scan()
    var acls []Acl
    for scanner.Scan() {
        fields := strings.Split(scanner.Text(), "\t")

        acl := Acl{App: fields[0], Port: fields[1]}
        acls = append(acls, acl)

        if acl.Port == "80" {
            printBackend(fields[2:], acl)
        } else {
            printListen(fields[2:], acl)
        }

        fmt.Println()
    }

    printFrontend(acls)
}

func printFrontend(acls []Acl) {
    fmt.Printf(frontendStart + "\n")

    for _, acl := range acls {
        if acl.Port == "80" {
            strippedApp := strings.Replace(acl.App, "lauras-", "", -1)
            fmt.Printf(aclFormat + "\n", acl.App, strippedApp, domain, strippedApp, acl.App)
        }
    }

    fmt.Printf(frontendEnd + "\n", defaultBackend)
}

func printListen(servers []string, acl Acl) {
    fmt.Printf(listen + "\n", acl.App, acl.Port, acl.Port)

	printServers(servers, acl)
}

func printBackend(servers []string, acl Acl) {
    strippedapp := strings.Replace(acl.App, "lauras-", "", -1)
    fmt.Printf(backend + "\n", strippedapp)

	printServers(servers, acl)
}

func printServers(servers []string, acl Acl) {
	for index, servername := range servers {
		if servername != "" {
			fmt.Printf(server + "\n", acl.App, index + 1, servername)
		}
	}
}

func checkerr(err error) {
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
