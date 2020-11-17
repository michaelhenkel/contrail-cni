.PHONY: build install run apply stop

build: client server

client: contrailcni_client/client.go
	GOOS=linux go build contrailcni_client/client.go

server: contrailcni_server/server.go 
	GOOS=linux go build contrailcni_server/server.go 

contrailcni_client=/opt/cni/bin/contrail-k8s-cni
$(contrailcni_client): client
	mkdir -p $$(dirname $(contrailcni_client))
	cp client $(contrailcni_client)
	
contrailcni_server=/opt/cni/bin/contrail-k8s-cni-server
$(contrailcni_server): server
	mkdir -p $$(dirname $(contrailcni_server))
	cp server $(contrailcni_server)

install: $(contrailcni_client) $(contrailcni_server)
	mkdir -p /etc/cni/net.d/
	cp 10-contrail.conf /etc/cni/net.d/

export KUBECONFIG ?= $(HOME)/.kube/config
run-only:
	mkdir -p /var/run/contrail
	pgrep -f '^$(contrailcni_server) ' || (nohup $(contrailcni_server) --incluster=false --kubeconfig=$(KUBECONFIG) >contrail-k8s-cni-server.out &)
	wait
run: install run-only

stop:
	pkill -f '^$(contrailcni_server) ' || true
	rm -f ./server contrail-k8s-cni-server.out

