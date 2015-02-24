.SILENT :
.PHONY : hud clean fmt test

TAG:=`git describe --abbrev=0 --tags --always`
LDFLAGS:=-X main.buildVersion `git describe --long`

all: hud

deps:
	glock sync github.com/jwilder/hud

hud:
	echo "Building hud"
	go install -ldflags "$(LDFLAGS)" github.com/jwilder/hud

clean: dist-clean
	rm -f $(GOPATH)/bin/hud

fmt:
	go fmt github.com/jwilder/hud/...

test:
	go test -v github.com/jwilder/hud/...

dist-clean:
	rm -rf dist
	rm -f hud-*.tar.gz

dist-init:
	mkdir -p dist/$$GOOS/$$GOARCH

dist-build: dist-init
	echo "Compiling $$GOOS/$$GOARCH"
	go build -ldflags "$(LDFLAGS)" -o dist/$$GOOS/$$GOARCH/hud github.com/jwilder/hud

dist-linux-amd64:
	export GOOS="linux"; \
	export GOARCH="amd64"; \
	$(MAKE) dist-build

dist-linux-386:
	export GOOS="linux"; \
	export GOARCH="386"; \
	$(MAKE) dist-build

dist-darwin-amd64:
	export GOOS="darwin"; \
	export GOARCH="amd64"; \
	$(MAKE) dist-build

dist-darwin-386:
	export GOOS="darwin"; \
	export GOARCH="386"; \
	$(MAKE) dist-build

dist: dist-clean dist-init dist-linux-amd64 dist-linux-386 dist-darwin-amd64 dist-darwin-386

release-tarball:
	echo "Building $$GOOS-$$GOARCH-$(TAG).tar.gz"
	GZIP=-9 tar -cvzf hud-$$GOOS-$$GOARCH-$(TAG).tar.gz -C dist/$$GOOS/$$GOARCH hud >/dev/null 2>&1

release-linux-amd64:
	export GOOS="linux"; \
	export GOARCH="amd64"; \
	$(MAKE) release-tarball

release-linux-386:
	export GOOS="linux"; \
	export GOARCH="386"; \
	$(MAKE) release-tarball

release-darwin-amd64:
	export GOOS="darwin"; \
	export GOARCH="amd64"; \
	$(MAKE) release-tarball

release-darwin-386:
	export GOOS="darwin"; \
	export GOARCH="386"; \
	$(MAKE) release-tarball

release: deps dist release-linux-amd64 release-linux-386 release-darwin-amd64 release-darwin-386
