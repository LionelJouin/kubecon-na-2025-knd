REGISTRY ?= localhost:5000/kubecon-na-2025-knd
VERSION ?= $(shell git describe --dirty --tags --always 2>/dev/null)

all: build-image

.PHONY: .build-image
build-image:
	docker build -t kubecon-na-2025-knd:$(VERSION) -f ./build/Dockerfile .

.PHONY: push-image
push-image: build-image
	docker tag kubecon-na-2025-knd:$(VERSION) $(REGISTRY)/kubecon-na-2025-knd:$(VERSION)
	docker push $(REGISTRY)/kubecon-na-2025-knd:$(VERSION)