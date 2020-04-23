all: generate deps

deps:
	go get -d -v

gen: generate

generate:
	go generate

build: generate deps
	go build

run: build
	./creamy-stuff

install: generate deps
	go install

uninstall:
	go clean -x -i

clean:
	rm -rf templates/*.qtpl.go
	go clean -x

image:
	docker build -t albinodrought/creamy-stuff .
