package stack

import "testing"

func TestEvent_RequestMethod_ResponseStatus(t *testing.T) {
	var e Event
	if e.RequestMethod() != "" || e.ResponseStatus() != 0 {
		t.Fatal("empty event")
	}
	e.Request = &Message{Method: MethodInvite, IsRequest: true}
	if e.RequestMethod() != MethodInvite {
		t.Fatalf("got %q", e.RequestMethod())
	}
	e.Response = &Message{IsRequest: false, StatusCode: 180, StatusText: "Ringing"}
	if e.ResponseStatus() != 180 {
		t.Fatalf("got %d", e.ResponseStatus())
	}
}
