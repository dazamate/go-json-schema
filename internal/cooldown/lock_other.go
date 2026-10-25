//go:build !unix && !windows

package cooldown

import (
	"errors"
	"os"
)

// Without file locks the cooldown holds within this process only.
func tryLock(*os.File) (bool, error) { return false, errors.ErrUnsupported }

func unlock(*os.File) {}
