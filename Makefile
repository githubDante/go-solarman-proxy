.PHONY: default all amd64 arm clean

TARGET=go-solarmanV5-proxy


default: amd64
all: arm amd64


amd64:
	$(shell mkdir -p build/x64)
	GOARCH=amd64 go build -ldflags "-w -s" -o build/x64/${TARGET}

arm:
	$(shell mkdir -p build/arm)
	GOARCH=arm go build -ldflags "-w -s" -o build/arm/${TARGET}


clean:
	$(shell rm -rf build)
	@echo -ne

