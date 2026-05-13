package stack

// RequestMethod returns the SIP method for events that carry a request (e.g. EventRequestReceived).
// Empty string if Request is nil.
func (e Event) RequestMethod() string {
	if e.Request == nil {
		return ""
	}
	return e.Request.Method
}

// ResponseStatus returns the SIP status code for events that carry a response (e.g. EventResponseReceived).
// Zero if Response is nil.
func (e Event) ResponseStatus() int {
	if e.Response == nil {
		return 0
	}
	return e.Response.StatusCode
}
