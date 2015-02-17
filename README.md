Marathon HaProxy Subdomain Bridge
===

Why?
====

When deploying an app 'myapp' to Marathon we want to access it at myapp.domain.tld

How?
====

The bridge.go program uses Marathon's tasks API to regerenate HaProxy config. It is called by the haproxy.cron every minute and then the 
refresh-haproxy script moves the config into /etc/haproxy/haproxy.cfg and reloads HaProxy.

Install
====

Run the install-bridge.sh script to install the scripts + cron file to machines in the Mesos cluster.








