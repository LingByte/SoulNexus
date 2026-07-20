package callbinding

import "testing"

func TestUserUtterancePCM_SlidingWindow(t *testing.T) {
	const callID = "test-utterance-pcm"
	ClearUserUtterancePCM(callID)
	defer ClearUserUtterancePCM(callID)

	chunk := make([]byte, 4000)
	for i := 0; i < 100; i++ {
		AppendUserUtterancePCM(callID, chunk, 16000)
	}
	pcm, sr, ok := GetUserUtterancePCM(callID)
	if !ok {
		t.Fatal("expected buffered pcm")
	}
	if sr != 16000 {
		t.Fatalf("sampleRate=%d", sr)
	}
	if len(pcm) > maxUtterancePCMBytes {
		t.Fatalf("pcm len %d exceeds cap", len(pcm))
	}
	if len(pcm) != maxUtterancePCMBytes && len(pcm) != maxUtterancePCMBytes-1 {
		// Cap trims to max; odd-byte fix may drop one.
		if len(pcm) < maxUtterancePCMBytes-4000 {
			t.Fatalf("pcm len %d unexpectedly small", len(pcm))
		}
	}
}
