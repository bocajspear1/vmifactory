.PHONY: all
all: web runner

.PHONY: web
web:	
	go build -o vmif-web cmd/vmif-web/main.go

.PHONY: runner
runner:	
	go build -o vmif-run cmd/vmif-run/main.go