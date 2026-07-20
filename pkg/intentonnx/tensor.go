package intentonnx

import (
	"fmt"

	ort "github.com/yalue/onnxruntime_go"
)

func concreteShape(dim ort.Shape, seq int, batch int) (ort.Shape, error) {
	if seq <= 0 {
		return nil, fmt.Errorf("seq must be > 0")
	}
	if batch <= 0 {
		batch = 1
	}
	out := make(ort.Shape, len(dim))
	var unknown []int
	for i, d := range dim {
		if d <= 0 {
			unknown = append(unknown, i)
		}
	}
	if len(dim) == 2 && len(unknown) == 2 {
		out[0] = int64(batch)
		out[1] = int64(seq)
		return out, nil
	}
	for i, d := range dim {
		if d > 0 {
			out[i] = d
			continue
		}
		if len(dim) == 2 && len(unknown) == 1 {
			if unknown[0] == 0 {
				out[i] = int64(batch)
			} else {
				out[i] = int64(seq)
			}
			continue
		}
		out[i] = int64(seq)
	}
	return out, nil
}

func intsToInt64Row(dst []int64, src []int) {
	if len(dst) != len(src) {
		m := len(dst)
		if len(src) < m {
			m = len(src)
		}
		for i := 0; i < m; i++ {
			dst[i] = int64(src[i])
		}
		return
	}
	for i := range src {
		dst[i] = int64(src[i])
	}
}

func intsToInt32Row(dst []int32, src []int) {
	if len(dst) != len(src) {
		m := len(dst)
		if len(src) < m {
			m = len(src)
		}
		for i := 0; i < m; i++ {
			dst[i] = int32(src[i])
		}
		return
	}
	for i := range src {
		dst[i] = int32(src[i])
	}
}
