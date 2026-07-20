package knowledge

// Hit is one retrieved chunk from the bound knowledge base.
type Hit struct {
	Title    string
	Content  string
	Source   string
	Score    float64
	RecordID string
	ChunkID  uint
	Quoted   bool
}

// Binding describes the per-call knowledge base binding.
type Binding struct {
	NamespaceID  uint
	Collection   string
	Enabled      bool
	AssistantID  uint
	SearchConfig SearchConfig
}

// SearchConfig is per-call knowledge retrieval tuning (from assistant agent_config).
type SearchConfig struct {
	TopK                   int
	MinScore               float64
	Threshold              float64
	UseMemoEnhanceQuery    bool
	UsePreviousRoundsSlice int
	AutoEnrich             bool
}
