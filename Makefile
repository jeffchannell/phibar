BUILDDIR=./build
PACKAGE=github.com/jeffchannell/phibar/main
VERSION=`git describe --abbrev=0 --tags`'-'`git rev-parse --short HEAD`

all: init resources linux

clean:
	rm -rf ${BUILDDIR}

init:
	go get github.com/atotto/clipboard
	go install github.com/atotto/clipboard
	go get github.com/golang/freetype/truetype
	go get github.com/hajimehoshi/ebiten
	go get github.com/jeffchannell/golden
	go get golang.org/x/image/font
	mkdir -p ${BUILDDIR}

linux:
	GOOS=linux GOARCH=amd64 go build -o ${BUILDDIR}/phibar.x86_64.linux ${PACKAGE}

resources:
	go generate github.com/jeffchannell/phibar/main/resources/