PKG_PREFIX := github.com/VictoriaMetrics/VictoriaMetrics

DATEINFO_TAG ?= $(shell date -u +'%Y%m%d-%H%M%S')
BUILDINFO_TAG ?= $(shell echo $$(git describe --long --all | tr '/' '-')$$( \
	      git diff-index --quiet HEAD -- || echo '-dirty-'$$(git diff-index -u HEAD | openssl sha1 | cut -d' ' -f2 | cut -c 1-8)))
LATEST_TAG ?= cluster-latest

PKG_TAG ?= $(shell git tag -l --points-at HEAD)
ifeq ($(PKG_TAG),)
PKG_TAG := $(BUILDINFO_TAG)
endif

GO_BUILDINFO = -X '$(PKG_PREFIX)/lib/buildinfo.Version=$(APP_NAME)-$(DATEINFO_TAG)-$(BUILDINFO_TAG)'

.PHONY: $(MAKECMDGOALS)

include app/*/Makefile
include deployment/*/Makefile
include package/release/Makefile

all: \
	vminsert \
	vmselect \
	vmstorage

all-pure: \
	vminsert-pure \
	vmselect-pure \
	vmstorage-pure

clean:
	rm -rf bin/*

vmcluster-linux-amd64: \
	vminsert-linux-amd64 \
	vmselect-linux-amd64 \
	vmstorage-linux-amd64

vmcluster-linux-arm64: \
	vminsert-linux-arm64 \
	vmselect-linux-arm64 \
	vmstorage-linux-arm64

vmcluster-linux-arm: \
	vminsert-linux-arm \
	vmselect-linux-arm \
	vmstorage-linux-arm

vmcluster-linux-ppc64le: \
	vminsert-linux-ppc64le \
	vmselect-linux-ppc64le \
	vmstorage-linux-ppc64le

vmcluster-linux-386: \
	vminsert-linux-386 \
	vmselect-linux-386 \
	vmstorage-linux-386

vmcluster-freebsd-amd64: \
	vminsert-freebsd-amd64 \
	vmselect-freebsd-amd64 \
	vmstorage-freebsd-amd64

vmcluster-openbsd-amd64: \
	vminsert-openbsd-amd64 \
	vmselect-openbsd-amd64 \
	vmstorage-openbsd-amd64

vmcluster-windows-amd64: \
	vminsert-windows-amd64 \
	vmselect-windows-amd64 \
	vmstorage-windows-amd64

vmcluster-crossbuild: \
	vmcluster-linux-amd64 \
	vmcluster-linux-arm64 \
	vmcluster-linux-arm \
	vmcluster-linux-ppc64le \
	vmcluster-linux-386 \
	vmcluster-freebsd-amd64 \
	vmcluster-openbsd-amd64

publish: package-base \
	publish-vminsert \
	publish-vmselect \
	publish-vmstorage

package: \
	package-vminsert \
	package-vmselect \
	package-vmstorage

publish-release:
	git checkout $(TAG) && LATEST_TAG=stable $(MAKE) release publish && \
		git checkout $(TAG)-cluster && LATEST_TAG=cluster-stable $(MAKE) release publish && \
		git checkout $(TAG)-enterprise && LATEST_TAG=enterprise-stable $(MAKE) release publish && \
		git checkout $(TAG)-enterprise-cluster && LATEST_TAG=enterprise-cluster-stable $(MAKE) release publish

release: \
	release-vmcluster

release-vmcluster: \
	release-vmcluster-linux-amd64 \
	release-vmcluster-linux-arm64 \
	release-vmcluster-freebsd-amd64 \
	release-vmcluster-openbsd-amd64 \
	release-vmcluster-windows-amd64

release-vmcluster-linux-amd64:
	GOOS=linux GOARCH=amd64 $(MAKE) release-vmcluster-goos-goarch

release-vmcluster-linux-arm64:
	GOOS=linux GOARCH=arm64 $(MAKE) release-vmcluster-goos-goarch

release-vmcluster-freebsd-amd64:
	GOOS=freebsd GOARCH=amd64 $(MAKE) release-vmcluster-goos-goarch

release-vmcluster-openbsd-amd64:
	GOOS=openbsd GOARCH=amd64 $(MAKE) release-vmcluster-goos-goarch

release-vmcluster-windows-amd64:
	GOARCH=amd64 $(MAKE) release-vmcluster-windows-goarch

release-vmcluster-goos-goarch: \
	vminsert-$(GOOS)-$(GOARCH)-prod \
	vmselect-$(GOOS)-$(GOARCH)-prod \
	vmstorage-$(GOOS)-$(GOARCH)-prod
	cd bin && \
		tar --transform="flags=r;s|-$(GOOS)-$(GOARCH)||" -czf victoria-metrics-$(GOOS)-$(GOARCH)-$(PKG_TAG).tar.gz \
			vminsert-$(GOOS)-$(GOARCH)-prod \
			vmselect-$(GOOS)-$(GOARCH)-prod \
			vmstorage-$(GOOS)-$(GOARCH)-prod \
		&& sha256sum victoria-metrics-$(GOOS)-$(GOARCH)-$(PKG_TAG).tar.gz \
			vminsert-$(GOOS)-$(GOARCH)-prod \
			vmselect-$(GOOS)-$(GOARCH)-prod \
			vmstorage-$(GOOS)-$(GOARCH)-prod \
			| sed s/-$(GOOS)-$(GOARCH)-prod/-prod/ > victoria-metrics-$(GOOS)-$(GOARCH)-$(PKG_TAG)_checksums.txt
	cd bin && rm -rf \
		vminsert-$(GOOS)-$(GOARCH)-prod \
		vmselect-$(GOOS)-$(GOARCH)-prod \
		vmstorage-$(GOOS)-$(GOARCH)-prod

release-vmcluster-windows-goarch: \
	vminsert-windows-$(GOARCH)-prod \
	vmselect-windows-$(GOARCH)-prod \
	vmstorage-windows-$(GOARCH)-prod
	cd bin && \
		zip victoria-metrics-windows-$(GOARCH)-$(PKG_TAG).zip \
			vminsert-windows-$(GOARCH)-prod.exe \
			vmselect-windows-$(GOARCH)-prod.exe \
			vmstorage-windows-$(GOARCH)-prod.exe \
		&& sha256sum victoria-metrics-windows-$(GOARCH)-$(PKG_TAG).zip \
			vminsert-windows-$(GOARCH)-prod.exe \
			vmselect-windows-$(GOARCH)-prod.exe \
			vmstorage-windows-$(GOARCH)-prod.exe \
		> victoria-metrics-windows-$(GOARCH)-$(PKG_TAG)_checksums.txt
	cd bin && rm -rf \
		vminsert-windows-$(GOARCH)-prod.exe \
		vmselect-windows-$(GOARCH)-prod.exe \
		vmstorage-windows-$(GOARCH)-prod.exe

pprof-cpu:
	go tool pprof -trim_path=github.com/VictoriaMetrics/VictoriaMetrics@ $(PPROF_FILE)

fmt:
	gofmt -l -w -s ./lib
	gofmt -l -w -s ./app

vet:
	go vet ./lib/...
	go vet ./app/...

check-all: fmt vet golangci-lint govulncheck

test:
	go test ./lib/... ./app/...

test-race:
	go test -race ./lib/... ./app/...

test-pure:
	CGO_ENABLED=0 go test ./lib/... ./app/...

test-full:
	go test -coverprofile=coverage.txt -covermode=atomic ./lib/... ./app/...

test-full-386:
	GOARCH=386 go test -coverprofile=coverage.txt -covermode=atomic ./lib/... ./app/...

benchmark:
	go test -bench=. ./lib/...
	go test -bench=. ./app/...

benchmark-pure:
	CGO_ENABLED=0 go test -bench=. ./lib/...
	CGO_ENABLED=0 go test -bench=. ./app/...

vendor-update:
	go get -u -d ./lib/...
	go get -u -d ./app/...
	go mod tidy -compat=1.19
	go mod vendor

app-local:
	CGO_ENABLED=1 go build $(RACE) -ldflags "$(GO_BUILDINFO)" -o bin/$(APP_NAME)$(RACE) $(PKG_PREFIX)/app/$(APP_NAME)

app-local-pure:
	CGO_ENABLED=0 go build $(RACE) -ldflags "$(GO_BUILDINFO)" -o bin/$(APP_NAME)-pure$(RACE) $(PKG_PREFIX)/app/$(APP_NAME)

app-local-goos-goarch:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(RACE) -ldflags "$(GO_BUILDINFO)" -o bin/$(APP_NAME)-$(GOOS)-$(GOARCH)$(RACE) $(PKG_PREFIX)/app/$(APP_NAME)

app-local-windows-goarch:
	CGO_ENABLED=0 GOOS=windows GOARCH=$(GOARCH) go build $(RACE) -ldflags "$(GO_BUILDINFO)" -o bin/$(APP_NAME)-windows-$(GOARCH)$(RACE).exe $(PKG_PREFIX)/app/$(APP_NAME)

quicktemplate-gen: install-qtc
	qtc

install-qtc:
	which qtc || go install github.com/valyala/quicktemplate/qtc@latest


golangci-lint: install-golangci-lint
	golangci-lint run

install-golangci-lint:
	which golangci-lint || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.51.2

govulncheck: install-govulncheck
	govulncheck ./...

install-govulncheck:
	which govulncheck || go install golang.org/x/vuln/cmd/govulncheck@latest

install-wwhrd:
	which wwhrd || go install github.com/frapposelli/wwhrd@latest

check-licenses: install-wwhrd
	wwhrd check -f .wwhrd.yml

copy-docs:
	echo "---" > ${DST}
	@if [ ${ORDER} -ne 0 ]; then \
		echo "sort: ${ORDER}" >> ${DST}; \
		echo "weight: ${ORDER}" >> ${DST}; \
		echo "menu:\n  docs:\n    parent: 'victoriametrics'\n    weight: ${ORDER}" >> ${DST}; \
	fi

	echo "title: ${TITLE}" >> ${DST}
	@if [ ${OLD_URL} ]; then \
		echo "aliases:\n  - ${OLD_URL}" >> ${DST}; \
	fi
	echo "---" >> ${DST}
	cat ${SRC} >> ${DST}
	sed -i='.tmp' 's/<img src=\"docs\//<img src=\"/' ${DST}
	rm -rf docs/*.tmp

# Copies docs for all components and adds the order/weight tag, title, menu position and alias with the backward compatible link for the old site.
# For ORDER=0 it adds no order tag/weight tag.
# FOR OLD_URL - relative link, used for backward compatibility with the link from documentation based on GitHub pages (old one)
# FOR OLD_URL='' it adds no alias, it should be empty for every new page, don't change it for already existing links.
# Images starting with <img src="docs/ are replaced with <img src="
# Cluster docs are supposed to be ordered as 2nd.
# The rest of docs is ordered manually.
docs-sync:
	SRC=README.md DST=docs/Cluster-VictoriaMetrics.md OLD_URL='/Cluster-VictoriaMetrics.html' ORDER=2 TITLE='Cluster version' $(MAKE) copy-docs
	SRC=app/vmagent/README.md DST=docs/vmagent.md OLD_URL='/vmagent.html' ORDER=3 TITLE=vmagent $(MAKE) copy-docs
	SRC=app/vmalert/README.md DST=docs/vmalert.md OLD_URL='/vmalert.html' ORDER=4 TITLE=vmalert $(MAKE) copy-docs
	SRC=app/vmauth/README.md DST=docs/vmauth.md OLD_URL='/vmauth.html' ORDER=5 TITLE=vmauth $(MAKE) copy-docs
	SRC=app/vmbackup/README.md DST=docs/vmbackup.md OLD_URL='/vmbackup.html' ORDER=6 TITLE=vmbackup $(MAKE) copy-docs
	SRC=app/vmrestore/README.md DST=docs/vmrestore.md OLD_URL='/vmrestore.html' ORDER=7 TITLE=vmrestore $(MAKE) copy-docs
	SRC=app/vmctl/README.md DST=docs/vmctl.md OLD_URL='/vmctl.html' ORDER=8 TITLE=vmctl $(MAKE) copy-docs
	SRC=app/vmgateway/README.md DST=docs/vmgateway.md OLD_URL='/vmgateway.html' ORDER=9 TITLE=vmgateway $(MAKE) copy-docs
	SRC=app/vmbackupmanager/README.md DST=docs/vmbackupmanager.md OLD_URL='/vmbackupmanager.html' ORDER=10 TITLE=vmbackupmanager $(MAKE) copy-docs
