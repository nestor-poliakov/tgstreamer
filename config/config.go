package config

import (
	_ "embed"
)

//go:embed reference.yaml
var Reference []byte
