//go:build tools

package hack

import (
	_ "k8s.io/code-generator"
	_ "k8s.io/code-generator/cmd/deepcopy-gen"
)
