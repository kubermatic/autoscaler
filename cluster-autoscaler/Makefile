all: build

TAG?=dev
FLAGS=
ENVVAR=
GOOS?=linux
REGISTRY?=staging-k8s.gcr.io
BASEIMAGE?=k8s.gcr.io/debian-base-amd64:0.3.2

CLUSTERAPI_VENDOR_DIR = $(GOPATH)/src/k8s.io/autoscaler/cluster-autoscaler/cloudprovider/clusterapi/internal/vendor

deps:
	go get -v github.com/tools/godep

build: clean deps | $(CLUSTERAPI_VENDOR_DIR)
	$(ENVVAR) GOOS=$(GOOS) godep go build ./...
	$(ENVVAR) GOOS=$(GOOS) godep go build -o cluster-autoscaler

build-binary: clean deps | $(CLUSTERAPI_VENDOR_DIR)
	$(ENVVAR) GOOS=$(GOOS) godep go build -o cluster-autoscaler

test-unit: clean deps build
	$(ENVVAR) godep go test --test.short -race ./... $(FLAGS)

dev-release: build-binary execute-release
	echo "Release ${TAG} completed"

make-image:
	docker build --pull --build-arg BASEIMAGE=${BASEIMAGE} \
	    -t ${REGISTRY}/cluster-autoscaler:${TAG} .

push-image:
	./push_image.sh ${REGISTRY}/cluster-autoscaler:${TAG}

execute-release: make-image push-image

clean:
	rm -f cluster-autoscaler

generate:
	go generate ./cloudprovider/aws

format:
	test -z "$$(find . -path ./vendor -prune -type f -o -name '*.go' -exec gofmt -s -d {} + | tee /dev/stderr)" || \
    test -z "$$(find . -path ./vendor -prune -type f -o -name '*.go' -exec gofmt -s -w {} + | tee /dev/stderr)"

docker-builder:
	docker build -t autoscaling-builder ../builder

build-in-docker: clean docker-builder
	docker run -v `pwd`:/gopath/src/k8s.io/autoscaler/cluster-autoscaler/ autoscaling-builder:latest bash -c 'cd /gopath/src/k8s.io/autoscaler/cluster-autoscaler && make build-binary'

release: build-in-docker execute-release
	echo "Full in-docker release ${TAG} completed"

container: build-in-docker make-image
	echo "Created in-docker image ${TAG}"

test-in-docker: clean docker-builder
	docker run -v `pwd`:/gopath/src/k8s.io/autoscaler/cluster-autoscaler/ autoscaling-builder:latest bash -c 'cd /gopath/src/k8s.io/autoscaler/cluster-autoscaler && godep go test ./... '

.PHONY: all deps build test-unit clean format execute-release dev-release docker-builder build-in-docker release generate

rclean: clean
	$(RM) -r $(CLUSTERAPI_VENDOR_DIR)

$(CLUSTERAPI_VENDOR_DIR):
	$(MAKE) vendor-clusterapi

.PHONY: vendor-clusterapi

vendor-clusterapi:
	go generate k8s.io/autoscaler/cluster-autoscaler/cloudprovider/clusterapi/internal/...
