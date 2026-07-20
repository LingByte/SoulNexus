package realtime

import (
	"context"
	"testing"

	"github.com/LingByte/lingllm/realtime"
)

type fakeGreetingAgent struct {
	creates int
}

func (f *fakeGreetingAgent) Start(context.Context) error { return nil }
func (f *fakeGreetingAgent) PushAudio([]byte) error      { return nil }
func (f *fakeGreetingAgent) CommitInputAudio() error     { return nil }
func (f *fakeGreetingAgent) CreateResponse(string) error {
	f.creates++
	return nil
}
func (f *fakeGreetingAgent) Cancel() error                   { return nil }
func (f *fakeGreetingAgent) Close() error                    { return nil }
func (f *fakeGreetingAgent) UpdateInstructions(string) error { return nil }

func TestTriggerProactiveGreetingServerVADAfterWelcomeSkipsCreate(t *testing.T) {
	ag := &fakeGreetingAgent{}
	TriggerProactiveGreeting(ag, "您好", nil, "c1", true, true)
	if ag.creates != 0 {
		t.Fatalf("creates=%d want 0 after welcome clip in server VAD mode", ag.creates)
	}
}

func TestTriggerProactiveGreetingServerVADNoWelcomeCreatesOnce(t *testing.T) {
	ag := &fakeGreetingAgent{}
	TriggerProactiveGreeting(ag, "您好", nil, "c2", true, false)
	if ag.creates != 1 {
		t.Fatalf("creates=%d want 1 when no welcome clip", ag.creates)
	}
}

func TestTriggerProactiveGreetingManualVADUsesSilenceCommit(t *testing.T) {
	ag := &fakeGreetingAgent{}
	TriggerProactiveGreeting(ag, "您好", nil, "c3", false, false)
	if ag.creates != 1 {
		t.Fatalf("creates=%d want 1 in manual VAD mode", ag.creates)
	}
}

var _ realtime.Agent = (*fakeGreetingAgent)(nil)
