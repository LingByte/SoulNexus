package intentonnx

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	ort "github.com/yalue/onnxruntime_go"
)

var (
	ortInitOnce sync.Once
	ortInitErr  error
	ortReady    atomic.Bool
)

// InitRuntime loads ONNX Runtime once per process. Call before [NewEngine].
func InitRuntime(sharedLibraryPath string) error {
	p := strings.TrimSpace(sharedLibraryPath)
	if p == "" {
		return fmt.Errorf("intentonnx: InitRuntime: empty shared library path")
	}
	ortInitOnce.Do(func() {
		ort.SetSharedLibraryPath(p)
		ortInitErr = ort.InitializeEnvironment()
		if ortInitErr == nil {
			ortReady.Store(true)
		}
	})
	return ortInitErr
}

// CloseRuntime releases the global ORT environment. Optional; call at process shutdown.
func CloseRuntime() error {
	ortReady.Store(false)
	return ort.DestroyEnvironment()
}

func requireRuntime() error {
	if !ortReady.Load() {
		return fmt.Errorf("intentonnx: call InitRuntime(sharedLibPath) before NewEngine")
	}
	return nil
}
