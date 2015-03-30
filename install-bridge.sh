#!/bin/bash

#
# Installs the Marathon HaProxy Subdomain Bridge on all nodes in the Mesos cluster
#

DOMAIN="example.com"
SERVERS="108.59.83.246 146.148.95.197 146.148.67.239 146.148.45.43 104.154.94.6"

install_bridge() {
  SERVER=$1
  echo "> Installing bridge on $SERVER"
  echo "> Transferring files"
  scp -o StrictHostKeyChecking=no bridge jclouds@$SERVER:~/haproxy-marathon-bridge
  scp -o StrictHostKeyChecking=no refresh-haproxy jclouds@$SERVER:~/refresh-haproxy
  scp -o StrictHostKeyChecking=no haproxycron jclouds@$SERVER:~/haproxycron
  echo "> Moving files to correct place"
  ssh -o StrictHostKeyChecking=no jclouds@$SERVER 'sudo mv -f ~/haproxy-marathon-bridge /usr/local/bin/haproxy-marathon-bridge; sudo mv -f ~/refresh-haproxy /usr/local/bin/refresh-haproxy; sudo mv -f ~/haproxycron /etc/cron.d/haproxycron; sudo chown root:root /usr/local/bin/haproxy-marathon-bridge /usr/local/bin/refresh-haproxy /etc/cron.d/haproxycron'
  echo "> DONE on $SERVER"
}

echo "> Building bridge.go"
go build bridge.go
echo "> Updating cronjob"
sed -i '' "s/<domain>/$DOMAIN/" haproxycron

for SERVER in $SERVERS
do
  install_bridge $SERVER $DOMAIN
done


