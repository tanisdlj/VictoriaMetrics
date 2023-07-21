GO_VERSION ?=1.20.6
SNAP_BUILDER_IMAGE := local/snap-vmctl-builder:2.0.0-$(shell echo $(GO_VERSION) | tr :/ __)

# truncate pkg_tag, snap has limitations for name length
SNAP_TAG := $(shell echo "$(PKG_TAG)" | cut -b1-31 )

snap-vmctl-builder-image:
	(docker image ls --format '{{.Repository}}:{{.Tag}}' | grep -q '$(SNAP_BUILDER_IMAGE)$$') \
	|| docker build \
		--build-arg GO_VERSION=$(GO_VERSION) \
		--tag $(SNAP_BUILDER_IMAGE) \
		snap/vmctl/local

build-vmctl-snap: snap-vmctl-builder-image
	docker run --rm \
		--mount type=bind,src="$(shell pwd)",dst=/builder \
		-w /builder \
		--env PKG_TAG=$(SNAP_TAG) \
		$(SNAP_BUILDER_IMAGE) \
		bash -c 'snapcraft clean && snapcraft snap -o "bin/vmctl_$(SNAP_TAG)_multi.snap"'

release-vmctl-snap: snap-vmctl-builder-image build-vmctl-snap
	docker run --rm \
		--mount type=bind,src=${HOME}/.snap,dst=/root/.snap \
		--mount type=bind,src="$(shell pwd)",dst=/builder \
		-w /builder \
		--env PKG_TAG=$(SNAP_TAG) \
		$(SNAP_BUILDER_IMAGE) \
		bash -c 'snapcraft login --with /root/.snap/login.json && snapcraft upload "bin/vmctl_$(SNAP_TAG)_multi.snap" --release beta,edge,candidate,stable'