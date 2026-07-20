package audio

// JobSnapshot is a provider-normalized audio moderation poll result.
type JobSnapshot struct {
	Status     string  // WAITING | DOING | FINISHED | FAILED
	Suggestion string  // pass | review | block
	Label      string
	Score      float64
	Msg        string
	Error      string
	Raw        any
}

// AudioJobPoller optionally exposes typed polling (preferred over raw GetCensorResult).
type AudioJobPoller interface {
	PollCensorAudio(taskID string) (*JobSnapshot, error)
}

const (
	JobWaiting  = "WAITING"
	JobDoing    = "DOING"
	JobFinished = "FINISHED"
	JobFailed   = "FAILED"
)

const (
	SuggestionPass   = "pass"
	SuggestionReview = "review"
	SuggestionBlock  = "block"
)
