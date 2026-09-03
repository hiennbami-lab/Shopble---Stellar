//go:build shopble

package maincmd

import "shopble/project/shopble/cmd"

func init() {
	executor = cmd.Execute
}
