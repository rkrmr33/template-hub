package assets

import (
	"embed"
)

//go:embed swagger.json
var StaticFS embed.FS
