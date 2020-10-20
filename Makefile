all: build
.PHONY: all

# Include the library makefile
include $(addprefix ./vendor/github.com/openshift/build-machinery-go/make/, \
	golang.mk \
	targets/openshift/images.mk \
	targets/openshift/deps.mk \
)

build-image:
	podman build --squash -f Dockerfile.shodan -t quay.io/mfojtik/shodan:v0.1
.PHONY: build-image

build-base:
	podman build --squash -f Dockerfile.base -t quay.io/mfojtik/shodan:base

push-image:
	podman push quay.io/mfojtik/shodan:v0.1
