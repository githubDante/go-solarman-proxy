.PHONY: default all amd64 arm clean gh_arm gh_amd64 gh_all

TARGET=go-solarmanV5-proxy


default: amd64
all: arm amd64
gh_all: gh_amd64 gh_arm


amd64:
	$(shell mkdir -p build/x64)
	GOARCH=amd64 go build -ldflags "-w -s" -o build/x64/${TARGET}

arm:
	$(shell mkdir -p build/arm)
	GOARCH=arm go build -ldflags "-w -s" -o build/arm/${TARGET}

gh_arm:
	$(shell mkdir -p build/arm)
	GOARCH=arm go build -ldflags "-w -s" -o build/arm/${TARGET}.arm

gh_amd64:
	$(shell mkdir -p build/x64)
	GOARCH=amd64 go build -ldflags "-w -s" -o build/x64/${TARGET}.x64

clean:
	$(shell rm -rf build)
	@echo -ne

