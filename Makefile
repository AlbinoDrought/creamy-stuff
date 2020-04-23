all: deps generate

deps:
	go get -d -v

gen: generate

generate:
	go generate

build: deps generate
	go build

run: build
	./creamy-stuff

install: deps generate
	go install

uninstall:
	go clean -x -i

clean:
	rm -rf templates/*.qtpl.go
	go clean -x

image:
	docker build -t albinodrought/creamy-stuff .
