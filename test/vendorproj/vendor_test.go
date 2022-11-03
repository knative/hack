package vendorproj_test

import (
	"embed"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
)

//go:embed vendor/modules.txt
var modulesTxt embed.FS

func TestVendoring(t *testing.T) {
	bytes, err := fs.ReadFile(modulesTxt, "vendor/modules.txt")
	assert.NoError(t, err)
	assert.Contains(t, string(bytes), "github.com/stretchr/testify/assert")
}
