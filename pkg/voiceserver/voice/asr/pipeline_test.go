package asr

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/recognizer"
)

// fakeASR implements recognizer.TranscribeService.
type fakeASR struct {
	mu         sync.Mutex
	tr         recognizer.TranscribeResult
	er         recognizer.ProcessError
	connCalls  int32
	sentBytes  int32
	endCalled  int32
	stopCalled int32
	vendor     string
	connErr    error
	sendErr    error
	active     bool
}

func (f *fakeASR) Init(tr recognizer.TranscribeResult, er recognizer.ProcessError) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tr = tr
	f.er = er
}
func (f *fakeASR) Vendor() string                 { return f.vendor }
func (f *fakeASR) ConnAndReceive(dialogID string) error {
	atomic.AddInt32(&f.connCalls, 1)
	return f.connErr
}
func (f *fakeASR) Activity() bool    { return f.active }
func (f *fakeASR) RestartClient()    {}
func (f *fakeASR) SendAudioBytes(b []byte) error {
	atomic.AddInt32(&f.sentBytes, int32(len(b)))
	return f.sendErr
}
func (f *fakeASR) SendEnd() error  { atomic.AddInt32(&f.endCalled, 1); return nil }
func (f *fakeASR) StopConn() error { atomic.AddInt32(&f.stopCalled, 1); return nil }

func (f *fakeASR) pushText(t string, final bool) {
	f.mu.Lock()
	fn := f.tr
	f.mu.Unlock()
	if fn != nil {
		fn(t, final, 0, "")
	}
}
func (f *fakeASR) pushErr(e error, fatal bool) {
	f.mu.Lock()
	fn := f.er
	f.mu.Unlock()
	if fn != nil {
		fn(e, fatal)
	}
}

func TestPipelineLazyConnectAndFeed(t *testing.T) {
	fa := &fakeASR{vendor: "fake"}
	p, err := New(Options{ASR: fa, SampleRate: 16000, MinFeedBytes: 100})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if atomic.LoadInt32(&fa.connCalls) != 0 {
		t.Fatalf("connect must be lazy")
	}
	if err := p.ProcessPCM(context.Background(), make([]byte, 50)); err != nil {
		t.Fatalf("ProcessPCM: %v", err)
	}
	if atomic.LoadInt32(&fa.connCalls) != 1 {
		t.Fatalf("expected 1 connect, got %d", fa.connCalls)
	}
	// 50 < minFeed → buffered, no send yet.
	if atomic.LoadInt32(&fa.sentBytes) != 0 {
		t.Fatalf("premature send: %d", fa.sentBytes)
	}
	// Crossing threshold triggers flush.
	_ = p.ProcessPCM(context.Background(), make([]byte, 60))
	if atomic.LoadInt32(&fa.sentBytes) != 110 {
		t.Fatalf("expected 110 bytes sent, got %d", fa.sentBytes)
	}

	// Vendor.
	if p.Vendor() != "fake" {
		t.Fatalf("Vendor mismatch")
	}
	_ = p.Close()
	if atomic.LoadInt32(&fa.endCalled) != 1 || atomic.LoadInt32(&fa.stopCalled) != 1 {
		t.Fatalf("Close did not SendEnd+Stop")
	}
}

func TestPipelineResampleOnDifferentRate(t *testing.T) {
	fa := &fakeASR{}
	p, err := New(Options{ASR: fa, InputSampleRate: 8000, SampleRate: 16000, MinFeedBytes: 1})
	if err != nil {
		t.Fatal(err)
	}
	// 160 bytes at 8k → ~320 at 16k after resample.
	in := make([]byte, 160)
	if err := p.ProcessPCM(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&fa.sentBytes) == 0 {
		t.Fatalf("expected resampled bytes sent")
	}
	_ = p.Close()
}

func TestPipelineTextAndErrorCallbacks(t *testing.T) {
	fa := &fakeASR{}
	p, _ := New(Options{ASR: fa, SampleRate: 16000, MinFeedBytes: 1})
	var gotText string
	var gotFinal bool
	var gotErr error
	var gotFatal bool
	p.SetTextCallback(func(s string, f bool) { gotText = s; gotFinal = f })
	p.SetErrorCallback(func(e error, f bool) { gotErr = e; gotFatal = f })
	_ = p.ProcessPCM(context.Background(), []byte{0x00, 0x01})
	fa.pushText("hello", true)
	if gotText != "hello" || !gotFinal {
		t.Fatalf("text callback: %q final=%v", gotText, gotFinal)
	}
	fa.pushErr(errors.New("boom"), true)
	if gotErr == nil || !gotFatal {
		t.Fatalf("error callback")
	}
	_ = p.Close()
}

func TestPipelineRejectsBadOptions(t *testing.T) {
	if _, err := New(Options{SampleRate: 16000}); err == nil {
		t.Fatal("expected nil-ASR error")
	}
	if _, err := New(Options{ASR: &fakeASR{}}); err == nil {
		t.Fatal("expected bad-rate error")
	}
}

func TestPipelineFlushAndRestart(t *testing.T) {
	fa := &fakeASR{}
	p, _ := New(Options{ASR: fa, SampleRate: 16000, MinFeedBytes: 1000})
	_ = p.ProcessPCM(context.Background(), make([]byte, 100))
	if atomic.LoadInt32(&fa.sentBytes) != 0 {
		t.Fatalf("should be buffered")
	}
	if err := p.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if atomic.LoadInt32(&fa.sentBytes) != 100 {
		t.Fatalf("Flush did not send: %d", fa.sentBytes)
	}
	if err := p.Restart(context.Background()); err != nil {
		t.Fatalf("Restart: %v", err)
	}
	if atomic.LoadInt32(&fa.connCalls) != 2 {
		t.Fatalf("Restart should reconnect, got %d", fa.connCalls)
	}
	_ = p.Close()
}
