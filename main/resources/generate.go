//go:generate go run ./palette/generate.go
//go:generate pngcrush ./images/palette.png ./images/palette.c.png
//go:generate $GOBIN/file2byteslice -package=images -input=./images/palette.c.png -output=./images/palette.go -var=Palette_png
//go:generate rm ./images/palette.c.png
//go:generate gofmt -s -w ./images

package resources
