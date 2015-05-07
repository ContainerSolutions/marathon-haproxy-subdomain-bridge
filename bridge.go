package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
	"log"
	"os/exec"
	"io")

const (
	defaultBackend = "journey"

	help_text = `
      USAGE: bridge -domain=<example.com> -marathon=<marathon host:port> -server=<true|false>

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
  bind :443 ssl crt /etc/haproxy/site.pem
  redirect scheme https code 301 if !{ ssl_fc }`

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
	FetchTasks(marathon string) ([]byte, error)
}

type MarathonTaskFetcher struct {
}

func (f MarathonTaskFetcher) FetchTasks(marathon string) ([]byte, error) {
	client := http.Client{}
	client.Timeout = time.Duration(60 * time.Second)

	req, err := http.NewRequest("GET", "http://"+marathon+"/v2/tasks", nil)
	checkerr(err)

	req.Header.Add("Accept", "text/plain")
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
	App  string
	Port string
}

func main() {
	var domain = flag.String("domain", "", "Domain")
	var marathon = flag.String("marathon", "", "Marathon")
	var help = flag.Bool("help", false, help_text)
	var rest = flag.Bool("server", true, "Run as server");
	var port = flag.String("port", "8080", "Server port");
	var pidFile = flag.String("pidfile", "/var/run/haproxy.pid", "Path to PID file")
	var configFile = flag.String("config", "/etc/haproxy/haproxy.cfg", "Path to config file")
	var binary = flag.String("haproxy", "/usr/local/bin/haproxy", "Path to HaProxy")

	flag.Parse()

	if *domain == "" || *marathon == "" || *help {
		fmt.Println(help_text)
		os.Exit(0)
	}

	fetcher := MarathonTaskFetcher{}

	if *rest {
		fmt.Println("Starting Marathon HaProxy bridge server on port " + *port)
		startServer(fetcher, marathon, domain, pidFile, configFile, binary, port)
	} else {
		config := generateHaProxyConfig(fetcher, marathon, domain)
		fmt.Println(config)
		os.Exit(0);
	}
}

func startServer(fetcher MarathonTaskFetcher,
				 marathon *string,
				 domain *string,
				 pidFile *string,
                 configFile *string,
				 binary *string,
				 port *string) {
	// TODO: Check if request originates from Marathon host. If not, reject

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		config := generateHaProxyConfig(fetcher, marathon, domain)
		writeConfigFile(config, *configFile)
		reload(*binary, *pidFile, *configFile)
	})
	log.Fatal(http.ListenAndServe(":" + *port, nil))
}

func writeConfigFile(config string, configFile string) {
	file, err := os.Create(configFile)
	if err != nil {
		fmt.Println(err)
	}
	n, err := io.WriteString(file, config)
	if err != nil {
		fmt.Println("Could not write HaProxy configuration file")
		fmt.Println(n, err)
	}
	// TODO: Close file
}

func reload(binary string, pidFile string, configFile string) error {
	pid, err := ioutil.ReadFile(pidFile)
	if err != nil {
		fmt.Print("Could not read pid file", err)
		return err
	}

	/* 	Setup all the command line parameters so we get an executable similar to
        /usr/local/bin/haproxy -f resources/haproxy_new.cfg -p resources/haproxy-private.pid -sf 1234
    */
	arg0 := "-f"
	arg1 := configFile
	arg2 := "-p"
	arg3 := pidFile
	arg4 := "-D"
	arg5 := "-sf"
	arg6 := strings.Trim(string(pid),"\n")
	var cmd *exec.Cmd

	// If this is the first run, the PID value will be empty, otherwise it will be > 0
	if len(arg6) > 0 {
		cmd = exec.Command(binary, arg0, arg1 ,arg2, arg3, arg4, arg5, arg6)
	} else {
		cmd = exec.Command(binary, arg0, arg1 ,arg2, arg3, arg4)
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmdErr := cmd.Run()
	if cmdErr != nil {
		fmt.Println(cmdErr.Error())
		return cmdErr
	}
	// TODO: Replace with info?
	log.Print("HaProxy Reload: " + out.String() + string(pid))
	return nil
}

func generateHaProxyConfig(fetcher TasksFetcher, marathon *string, domain *string) string {
	var config = header

	contents, err := fetcher.FetchTasks(*marathon)
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
			config = config + generateListen(fields[2:], acl)
		}

		config = config + "\n"
	}

	return config + generateFrontend(acls, *domain)
}

func generateFrontend(acls []Acl, domain string) string {
	var config string

	config = fmt.Sprintf(frontendStart + "\n")

	for _, acl := range acls {
		if acl.Port == "80" {
			strippedApp := strings.Replace(acl.App, "lauras-", "", -1)
			config = config + fmt.Sprintf(aclFormat+"\n", acl.App, strippedApp, domain, strippedApp, acl.App)
		}
	}

	return config + fmt.Sprintf(frontendEnd+"\n", defaultBackend)
}

func generateListen(servers []string, acl Acl) string {
	var config string

	config = fmt.Sprintf(listen+"\n", acl.App, acl.Port, acl.Port)

	return config + generateServers(servers, acl)
}

func generateBackend(servers []string, acl Acl) string {
	strippedapp := strings.Replace(acl.App, "lauras-", "", -1)

	config := fmt.Sprintf(backend+"\n", strippedapp)

	return config + generateServers(servers, acl)
}

func generateServers(servers []string, acl Acl) string {
	var config string
	for index, servername := range servers {
		if servername != "" {
			config = config + fmt.Sprintf(server+"\n", acl.App, index+1, servername)
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
