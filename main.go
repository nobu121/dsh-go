package main

import (
	"embed"

	"dsh-go/internal/app"
)

//go:embed frontend/index.html
//go:embed build/appicon.png
//go:embed app.version
//go:embed dsh.version
var assets embed.FS

func main() {
	appVer, _ := assets.ReadFile("app.version")
	dshVer, _ := assets.ReadFile("dsh.version")
	app.SetEmbeddedVersions(string(appVer), string(dshVer))
	app.Run(assets)
}
