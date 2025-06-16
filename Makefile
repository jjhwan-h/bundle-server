.PHONY: test
test:
		go test -v ./...

.PHONY: build
build:
		go build -o ./opa_bundle_server

.PHONY: run
run:
		nohup ./opa_bundle_server serve -p 4001 > /var/lib/opa/log/server.log 2>&1 &