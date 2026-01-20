package adapter

import (
	"fmt"

	"samplive/internal/detect"
)

// For returns the adapter for a detected runtime kind.
func For(kind detect.Kind) (RuntimeAdapter, error) {
	switch kind {
	case detect.KindSAMP:
		return SAMPRuntime{}, nil
	case detect.KindOMP:
		return OMPRuntime{}, nil
	default:
		return nil, fmt.Errorf("no adapter for runtime kind %q", kind)
	}
}
