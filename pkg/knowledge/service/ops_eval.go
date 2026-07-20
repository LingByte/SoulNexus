package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/LingByte/SoulNexus/pkg/utils"
	llmretrieve "github.com/LingByte/lingllm/retrieve"
	llmeval "github.com/LingByte/lingllm/retrieve/eval"
	"strings"
	"sync"
	"time"
)

// RunRetrievalEval evaluates retrieval quality for a namespace using labeled samples.
func (s *Service) RunRetrievalEval(ctx context.Context, namespace string, samples []llmeval.Sample, opts llmeval.Options) (llmeval.Report, error) {
	if s == nil {
		return llmeval.Report{}, fmt.Errorf("knowledge service is not initialized")
	}
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return llmeval.Report{}, fmt.Errorf("namespace is required")
	}
	samples = llmeval.NormalizeSamples(samples, namespace)

	strategy := parseEvalStrategy(opts.Strategy, s.RetrieveStrategy())
	retriever, err := s.buildRetriever(namespace, strategy, opts.TopK, opts.MinScore, nil, false, false)
	if err != nil {
		return llmeval.Report{}, err
	}
	opts.Strategy = string(strategy)
	opts.Namespace = namespace
	return llmeval.Run(ctx, llmeval.StrategyRetriever{Inner: retriever}, samples, opts)
}

// UnmarshalEvalSamples parses eval dataset JSON (array or JSONL) with relevant_ids / relevantIds aliases.
func UnmarshalEvalSamples(raw []byte, out *[]llmeval.Sample) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return fmt.Errorf("empty samples")
	}
	if raw[0] == '[' {
		var flex []flexEvalSample
		if err := json.Unmarshal(raw, &flex); err != nil {
			return err
		}
		*out = normalizeFlexSamples(flex)
		return validateEvalSamples(*out)
	}
	// JSONL: one sample per line
	lines := strings.Split(string(raw), "\n")
	samples := make([]llmeval.Sample, 0, len(lines))
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var flex flexEvalSample
		if err := json.Unmarshal([]byte(line), &flex); err != nil {
			return fmt.Errorf("line %d: %w", i+1, err)
		}
		samples = append(samples, flex.toSample())
	}
	if len(samples) == 0 {
		return fmt.Errorf("empty samples")
	}
	*out = samples
	return validateEvalSamples(*out)
}

type flexEvalSample struct {
	Query        string   `json:"query"`
	Namespace    string   `json:"namespace,omitempty"`
	RelevantIDs  []string `json:"relevant_ids"`
	RelevantIDs2 []string `json:"relevantIds"`
	GoldAnswer   string   `json:"gold_answer,omitempty"`
	GoldAnswer2  string   `json:"goldAnswer,omitempty"`
}

func (f flexEvalSample) toSample() llmeval.Sample {
	ids := f.RelevantIDs
	if len(ids) == 0 {
		ids = f.RelevantIDs2
	}
	ga := f.GoldAnswer
	if ga == "" {
		ga = f.GoldAnswer2
	}
	return llmeval.Sample{
		Query:       strings.TrimSpace(f.Query),
		Namespace:   strings.TrimSpace(f.Namespace),
		RelevantIDs: ids,
		GoldAnswer:  ga,
	}
}

func normalizeFlexSamples(flex []flexEvalSample) []llmeval.Sample {
	out := make([]llmeval.Sample, 0, len(flex))
	for _, f := range flex {
		out = append(out, f.toSample())
	}
	return out
}

func validateEvalSamples(samples []llmeval.Sample) error {
	if len(samples) == 0 {
		return fmt.Errorf("empty samples")
	}
	for i, s := range samples {
		if strings.TrimSpace(s.Query) == "" {
			return fmt.Errorf("sample %d: query is required", i+1)
		}
		if len(s.RelevantIDs) == 0 {
			return fmt.Errorf("sample %d: relevant_ids is required", i+1)
		}
	}
	return nil
}

// CompareRetrievalStrategies runs evaluation for multiple strategies on the same dataset.
func (s *Service) CompareRetrievalStrategies(ctx context.Context, namespace string, strategies []llmretrieve.Strategy, samples []llmeval.Sample, opts llmeval.Options) ([]llmeval.Report, error) {
	if len(strategies) == 0 {
		strategies = []llmretrieve.Strategy{
			llmretrieve.StrategyVector,
			llmretrieve.StrategyKeyword,
			llmretrieve.StrategyHybrid,
		}
	}
	reports := make([]llmeval.Report, 0, len(strategies))
	for _, strategy := range strategies {
		runOpts := opts
		runOpts.Strategy = string(strategy)
		report, err := s.RunRetrievalEval(ctx, namespace, samples, runOpts)
		if err != nil {
			return nil, fmt.Errorf("strategy %s: %w", strategy, err)
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func parseEvalStrategy(requested, configured string) llmretrieve.Strategy {
	switch strings.ToLower(strings.TrimSpace(requested)) {
	case "vector", "keyword", "hybrid":
		return llmretrieve.Strategy(requested)
	case "":
		switch strings.ToLower(strings.TrimSpace(configured)) {
		case "keyword":
			return llmretrieve.StrategyKeyword
		case "hybrid":
			return llmretrieve.StrategyHybrid
		default:
			return llmretrieve.StrategyVector
		}
	default:
		return llmretrieve.StrategyVector
	}
}

// EvalJobStatus tracks async retrieval evaluation progress.
type EvalJobStatus string

const (
	EvalJobPending EvalJobStatus = "pending"
	EvalJobRunning EvalJobStatus = "running"
	EvalJobDone    EvalJobStatus = "done"
	EvalJobFailed  EvalJobStatus = "failed"
)

// EvalJob is one background eval run (in-memory, per process).
type EvalJob struct {
	ID         string          `json:"jobId"`
	Status     EvalJobStatus   `json:"status"`
	Result     json.RawMessage `json:"result,omitempty"`
	Error      string          `json:"error,omitempty"`
	CreatedAt  time.Time       `json:"createdAt"`
	FinishedAt *time.Time      `json:"finishedAt,omitempty"`
}

var (
	evalJobsMu sync.RWMutex
	evalJobs   = map[string]*EvalJob{}
)

// StartRetrievalEvalJob runs evaluation in the background and returns a job id.
func StartRetrievalEvalJob(s *Service, namespace string, samples []llmeval.Sample, opts llmeval.Options, compare bool) string {
	id := newEvalJobID()
	job := &EvalJob{ID: id, Status: EvalJobPending, CreatedAt: time.Now()}
	evalJobsMu.Lock()
	evalJobs[id] = job
	evalJobsMu.Unlock()
	go runEvalJob(s, job, namespace, samples, opts, compare)
	return id
}

// GetEvalJob returns job state by id.
func GetEvalJob(id string) (*EvalJob, bool) {
	evalJobsMu.RLock()
	defer evalJobsMu.RUnlock()
	j, ok := evalJobs[id]
	if !ok || j == nil {
		return nil, false
	}
	cp := *j
	return &cp, true
}

func newEvalJobID() string {
	if utils.SnowflakeUtil != nil {
		raw := uint64(utils.SnowflakeUtil.NextID()) & 0x7FFFFFFFFFFFFFFF
		if raw > 0 {
			return formatEvalJobID(raw)
		}
	}
	return formatEvalJobID(uint64(time.Now().UnixNano()))
}

func formatEvalJobID(v uint64) string {
	const digits = "0123456789"
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = digits[v%10]
		v /= 10
	}
	return string(buf[i:])
}

func setEvalJobStatus(job *EvalJob, status EvalJobStatus) {
	if job == nil {
		return
	}
	evalJobsMu.Lock()
	job.Status = status
	evalJobsMu.Unlock()
}

func finishEvalJob(job *EvalJob, result any, err error) {
	if job == nil {
		return
	}
	now := time.Now()
	evalJobsMu.Lock()
	defer evalJobsMu.Unlock()
	job.FinishedAt = &now
	if err != nil {
		job.Status = EvalJobFailed
		job.Error = err.Error()
		return
	}
	job.Status = EvalJobDone
	if result != nil {
		if b, e := json.Marshal(result); e == nil {
			job.Result = b
		}
	}
}

func runEvalJob(s *Service, job *EvalJob, namespace string, samples []llmeval.Sample, opts llmeval.Options, compare bool) {
	setEvalJobStatus(job, EvalJobRunning)
	ctx := context.Background()
	if compare {
		reports, err := s.CompareRetrievalStrategies(ctx, namespace, nil, samples, opts)
		finishEvalJob(job, map[string]any{"strategies": reports}, err)
		return
	}
	report, err := s.RunRetrievalEval(ctx, namespace, samples, opts)
	finishEvalJob(job, report, err)
}
