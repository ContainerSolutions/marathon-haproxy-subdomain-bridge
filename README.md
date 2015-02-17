Marathon HaProxy Subdomain Bridge
===

Why?
====

When deploying an app 'myapp' to Marathon we want to access it at myapp.domain.tld

How?
====

The bridge.go program, based on Marathon's HaProxy bridge bash script
scans the Marathon tasks API and generates HaProxy config which maps the app unto a subdomain.
This script is called from a cronjob and writes to /tmp/ha-proxy.cfg. A second cron job replaces
the HaProxy config if the contents are new.

Cronjob
=======
```bash
* * * * * root /usr/local/bin/haproxy-marathon-bridge $(cat /etc/haproxy-marathon-bridge/marathons) > /tmp/haproxy.cfg ; /usr/local/bin/refresh-haproxy
```





