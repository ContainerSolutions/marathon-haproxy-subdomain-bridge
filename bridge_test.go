package main

import (
	"testing"
	// "fmt"
)

const (
	tasks = `lauras-artifactory	80	10.215.78.113:31000
lauras-commit-tester	80	10.224.225.144:31004
lauras-elk	80	10.16.85.111:31006
lauras-elk	5000	10.16.85.111:31007
lauras-gitlab	10022	10.16.85.111:31008
lauras-gitlab	10080	10.16.85.111:31009
lauras-gitlab	10443	10.16.85.111:31010
lauras-gitlab-postgres	5432	10.108.159.39:31006
lauras-gitlab-redis	6379	10.224.225.144:31005
lauras-jenkins	80	10.215.78.113:31011
lauras-journey	80	10.16.85.111:31000	10.215.78.113:31007
lauras-journey	443	10.16.85.111:31001	10.215.78.113:31008
lauras-orchestration	8083	10.224.225.144:31000
lauras-simulator	80	10.224.225.144:31006
webserver	80	10.16.85.111:31002	10.215.78.113:31010
webserver	443	10.16.85.111:31003	10.215.78.113:31012`

	expected = `global
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
backend artifactory
  balance leastconn
  option httpclose
  option forwardfor
  server lauras-artifactory-1 10.215.78.113:31000 check

backend commit-tester
  balance leastconn
  option httpclose
  option forwardfor
  server lauras-commit-tester-1 10.224.225.144:31004 check

backend elk
  balance leastconn
  option httpclose
  option forwardfor
  server lauras-elk-1 10.16.85.111:31006 check

listen lauras-elk-5000
  bind 0.0.0.0:5000
  mode tcp
  option tcplog
  balance leastconn
  server lauras-elk-1 10.16.85.111:31007 check

listen lauras-gitlab-10022
  bind 0.0.0.0:10022
  mode tcp
  option tcplog
  balance leastconn
  server lauras-gitlab-1 10.16.85.111:31008 check

listen lauras-gitlab-10080
  bind 0.0.0.0:10080
  mode tcp
  option tcplog
  balance leastconn
  server lauras-gitlab-1 10.16.85.111:31009 check

listen lauras-gitlab-10443
  bind 0.0.0.0:10443
  mode tcp
  option tcplog
  balance leastconn
  server lauras-gitlab-1 10.16.85.111:31010 check

listen lauras-gitlab-postgres-5432
  bind 0.0.0.0:5432
  mode tcp
  option tcplog
  balance leastconn
  server lauras-gitlab-postgres-1 10.108.159.39:31006 check

listen lauras-gitlab-redis-6379
  bind 0.0.0.0:6379
  mode tcp
  option tcplog
  balance leastconn
  server lauras-gitlab-redis-1 10.224.225.144:31005 check

backend jenkins
  balance leastconn
  option httpclose
  option forwardfor
  server lauras-jenkins-1 10.215.78.113:31011 check

backend journey
  balance leastconn
  option httpclose
  option forwardfor
  server lauras-journey-1 10.16.85.111:31000 check
  server lauras-journey-2 10.215.78.113:31007 check

listen lauras-journey-443
  bind 0.0.0.0:443
  mode tcp
  option tcplog
  balance leastconn
  server lauras-journey-1 10.16.85.111:31001 check
  server lauras-journey-2 10.215.78.113:31008 check

listen lauras-orchestration-8083
  bind 0.0.0.0:8083
  mode tcp
  option tcplog
  balance leastconn
  server lauras-orchestration-1 10.224.225.144:31000 check

backend simulator
  balance leastconn
  option httpclose
  option forwardfor
  server lauras-simulator-1 10.224.225.144:31006 check

backend webserver
  balance leastconn
  option httpclose
  option forwardfor
  server webserver-1 10.16.85.111:31002 check
  server webserver-2 10.215.78.113:31010 check

listen webserver-443
  bind 0.0.0.0:443
  mode tcp
  option tcplog
  balance leastconn
  server webserver-1 10.16.85.111:31003 check
  server webserver-2 10.215.78.113:31012 check

frontend http-in
  bind :80
  bind :443 ssl crt /etc/haproxy/site.pem
  acl subdomain-lauras-artifactory hdr_dom(host) -i artifactory.laurasjourney.nl
  use_backend artifactory if subdomain-lauras-artifactory
  acl subdomain-lauras-commit-tester hdr_dom(host) -i commit-tester.laurasjourney.nl
  use_backend commit-tester if subdomain-lauras-commit-tester
  acl subdomain-lauras-elk hdr_dom(host) -i elk.laurasjourney.nl
  use_backend elk if subdomain-lauras-elk
  acl subdomain-lauras-jenkins hdr_dom(host) -i jenkins.laurasjourney.nl
  use_backend jenkins if subdomain-lauras-jenkins
  acl subdomain-lauras-journey hdr_dom(host) -i journey.laurasjourney.nl
  use_backend journey if subdomain-lauras-journey
  acl subdomain-lauras-simulator hdr_dom(host) -i simulator.laurasjourney.nl
  use_backend simulator if subdomain-lauras-simulator
  acl subdomain-webserver hdr_dom(host) -i webserver.laurasjourney.nl
  use_backend webserver if subdomain-webserver
  default_backend journey
`
)

type StubFetcher struct{}

func (f StubFetcher) FetchTasks(hostport string) ([]byte, error) {
	return []byte(tasks), nil
}

func TestHaProxyConfig(t *testing.T) {
	fetcher := StubFetcher{}
	var marathon = "10.23.45.56"
	var domain = "laurasjourney.nl"
	config := generateHaProxyConfig(fetcher, &marathon, &domain)
	if config != expected {
		// fmt.Printf(config)
		// fmt.Printf(expected)
		t.Fatal("Generated HaProxy config is not as expected!")
	}

}
