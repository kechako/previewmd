GOCMD = go
BINDIR = bin
TARGETNAME = previewmd
TARGET = $(BINDIR)/$(TARGETNAME)
ASSETSDIR = assets
ASSETS = $(shell find $(ASSETSDIR) -type f)

.PHONY: all deps-dev build-assets build install

all: build

dev-deps:
	${GOCMD} get -u github.com/jessevdk/go-assets-builder

assets.go: $(ASSETS)
	go-assets-builder -s /assets -v Assets -o assets.go assets

build-assets: assets.go

$(TARGET): *.go assets.go
	$(GOCMD) build -o $(TARGET)

build: $(TARGET)

install: *.go assets.go
	$(GOCMD) install
