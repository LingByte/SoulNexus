package pipeline

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
)

func TestNew_RejectsEmpty(t *testing.T) {
	_, err := New("p", nil)
	if err == nil {
		t.Fatal("expected error on empty stages")
	}
}

func TestNew_RejectsNilStage(t *testing.T) {
	_, err := New("p", []Stage{nil})
	if err == nil {
		t.Fatal("expected error on nil stage")
	}
}

func TestNew_RejectsEmptyName(t *testing.T) {
	_, err := New("p", []Stage{StageFunc{StageName: ""}})
	if err == nil {
		t.Fatal("expected error on empty stage name")
	}
}

func TestPipeline_PassThrough(t *testing.T) {
	p, err := New("test", []Stage{
		StageFunc{StageName: "noop"},
	})
	if err != nil {
		t.Fatal(err)
	}
	src := make(chan Frame, 4)
	src <- Frame{Kind: KindTextFinal, Text: "hello"}
	src <- Frame{Kind: KindTextFinal, Text: "world"}
	close(src)

	out, errs := p.Run(context.Background(), src, engine.NopLogger{})

	got := drainText(out)
	if err := Wait(errs); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if want := []string{"hello", "world"}; !equalStrings(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPipeline_TwoStageTransform(t *testing.T) {
	upper := StageFunc{
		StageName: "upper",
		Fn: func(ctx context.Context, in <-chan Frame, out chan<- Frame, lg engine.Logger) error {
			defer close(out)
			for f := range in {
				f.Text = strings.ToUpper(f.Text)
				select {
				case out <- f:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		},
	}
	tag := StageFunc{
		StageName: "tag",
		Fn: func(ctx context.Context, in <-chan Frame, out chan<- Frame, lg engine.Logger) error {
			defer close(out)
			for f := range in {
				f.Text = "[" + f.Text + "]"
				select {
				case out <- f:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		},
	}

	p, err := New("test", []Stage{upper, tag})
	if err != nil {
		t.Fatal(err)
	}

	src := make(chan Frame, 2)
	src <- Frame{Kind: KindTextFinal, Text: "hi"}
	src <- Frame{Kind: KindTextFinal, Text: "bye"}
	close(src)

	out, errs := p.Run(context.Background(), src, engine.NopLogger{})
	got := drainText(out)
	if err := Wait(errs); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	want := []string{"[HI]", "[BYE]"}
	if !equalStrings(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPipeline_StageError(t *testing.T) {
	bad := StageFunc{
		StageName: "bad",
		Fn: func(ctx context.Context, in <-chan Frame, out chan<- Frame, lg engine.Logger) error {
			defer close(out)
			// Drain in to avoid blocking source goroutine.
			for range in {
			}
			return errors.New("boom")
		},
	}
	p, err := New("test", []Stage{bad})
	if err != nil {
		t.Fatal(err)
	}
	src := make(chan Frame)
	close(src)

	out, errs := p.Run(context.Background(), src, engine.NopLogger{})
	for range out { // drain
	}
	err = Wait(errs)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected wrapped 'boom' error, got %v", err)
	}
}

func TestPipeline_ContextCancellation(t *testing.T) {
	var dropped atomic.Int32
	slow := StageFunc{
		StageName: "slow",
		Fn: func(ctx context.Context, in <-chan Frame, out chan<- Frame, lg engine.Logger) error {
			defer close(out)
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case f, ok := <-in:
					if !ok {
						return nil
					}
					select {
					case out <- f:
					case <-ctx.Done():
						dropped.Add(1)
						return ctx.Err()
					}
				}
			}
		},
	}
	p, err := New("test", []Stage{slow})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	src := make(chan Frame) // never closed, never sends — stage waits forever
	out, errs := p.Run(ctx, src, engine.NopLogger{})

	// Cancel after 50ms.
	time.AfterFunc(50*time.Millisecond, cancel)

	for range out {
	}
	// errs should close within reasonable time (stage exits on ctx).
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	select {
	case e, ok := <-errs:
		if !ok {
			return
		}
		if !errors.Is(e, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", e)
		}
		// drain remainder
		for range errs {
		}
	case <-deadline.C:
		t.Fatal("pipeline did not honour context cancellation in time")
	}
}

func TestFrame_KindHelpers(t *testing.T) {
	cases := []struct {
		f                    Frame
		audio, text, control bool
	}{
		{Frame{Kind: KindPCM}, true, false, false},
		{Frame{Kind: KindTextFinal}, false, true, false},
		{Frame{Kind: KindTextInterim}, false, true, false},
		{Frame{Kind: KindAIText}, false, true, false},
		{Frame{Kind: KindBargeIn}, false, false, true},
		{Frame{Kind: KindToolCall}, false, false, true},
		{Frame{Kind: KindAITextDone}, false, false, true},
		{Frame{Kind: KindUserHangup}, false, false, true},
	}
	for _, tc := range cases {
		if got := tc.f.IsAudio(); got != tc.audio {
			t.Errorf("Kind=%d IsAudio=%v want %v", tc.f.Kind, got, tc.audio)
		}
		if got := tc.f.IsText(); got != tc.text {
			t.Errorf("Kind=%d IsText=%v want %v", tc.f.Kind, got, tc.text)
		}
		if got := tc.f.IsControl(); got != tc.control {
			t.Errorf("Kind=%d IsControl=%v want %v", tc.f.Kind, got, tc.control)
		}
	}
}

// drainText reads all frames from c and returns their Text field in
// arrival order.
func drainText(c <-chan Frame) []string {
	var out []string
	for f := range c {
		out = append(out, f.Text)
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
