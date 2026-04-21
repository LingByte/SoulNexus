package scriptlisten

import "testing"

func TestSubscribeNotify(t *testing.T) {
	ch, cancel := Subscribe("call-a")
	defer cancel()
	Notify("call-a")
	<-ch
	Notify("other")
	select {
	case <-ch:
		t.Fatal("unexpected wake for different call id")
	default:
	}
}
