VERSION=0.1.1
IMAGE=kavatech/healthd:$(VERSION)
BUILD_FLAGS="-X main.APPVERSION=$(VERSION)"

build:
	go build -ldflags $(BUILD_FLAGS)

build-linux:
	GOOS=linux go build -ldflags $(BUILD_FLAGS)

run:
	./healthd -etcd 192.168.1.215:2379 -ca ../healthagent/certs/ca.crt -crt ../healthagent/certs/server.example.org.crt -key ../healthagent/certs/server.example.org.key -port 3443

docker-build: build-linux
	docker build -t $(IMAGE) .

docker-push: docker-build
	docker push $(IMAGE)