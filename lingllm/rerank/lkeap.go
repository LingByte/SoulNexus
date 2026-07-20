package rerank

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	lkeap "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/lkeap/v20240522"
)

const (
	lkeapRerankEndpoint     = "lkeap.tencentcloudapi.com"
	lkeapDefaultRegion      = "ap-guangzhou"
	lkeapDefaultRerankModel = "lke-reranker-base"
	lkeapMaxDocuments       = 60
)

// LKEAPRerankClient uses Tencent Cloud LKEAP RunRerank for document reranking.
type LKEAPRerankClient struct {
	Model  string
	client *lkeap.Client
}

// NewLKEAPRerankClient creates a new LKEAP reranker client.
func NewLKEAPRerankClient(cfg *RerankClientConfig) (*LKEAPRerankClient, error) {
	if cfg == nil {
		return nil, errors.New(ErrNilClient)
	}

	secretID := strings.TrimSpace(cfg.APIKey)
	secretKey := strings.TrimSpace(cfg.SecretKey)
	if secretKey == "" && cfg.ExtraConfig != nil {
		secretKey = strings.TrimSpace(cfg.ExtraConfig["secret_key"])
	}
	if secretID == "" || secretKey == "" {
		return nil, fmt.Errorf("secret_id and secret_key are required for LKEAP rerank (set APIKey and SecretKey)")
	}

	region := lkeapDefaultRegion
	if cfg.ExtraConfig != nil {
		if r := strings.TrimSpace(cfg.ExtraConfig["region"]); r != "" {
			region = r
		}
	}

	credential := common.NewCredential(secretID, secretKey)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = lkeapRerankEndpoint

	client, err := lkeap.NewClient(credential, region, cpf)
	if err != nil {
		return nil, fmt.Errorf("create LKEAP client: %w", err)
	}

	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = lkeapDefaultRerankModel
	}

	return &LKEAPRerankClient{
		Model:  model,
		client: client,
	}, nil
}

func (c *LKEAPRerankClient) Provider() string {
	return ProviderLKEAP
}

func (c *LKEAPRerankClient) Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error) {
	if c == nil {
		return nil, errors.New(ErrNilClient)
	}
	if strings.TrimSpace(query) == "" {
		return nil, errors.New(ErrEmptyQuery)
	}
	if len(documents) == 0 {
		return nil, errors.New(ErrEmptyDocuments)
	}
	if len(documents) > lkeapMaxDocuments {
		return nil, fmt.Errorf("LKEAP rerank supports at most %d documents, got %d", lkeapMaxDocuments, len(documents))
	}
	topN = normalizeTopN(topN, len(documents))

	req := lkeap.NewRunRerankRequest()
	req.Query = common.StringPtr(query)
	req.Docs = common.StringPtrs(documents)
	req.Model = common.StringPtr(c.Model)

	resp, err := c.client.RunRerankWithContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LKEAP RunRerank: %w", err)
	}
	if resp == nil || resp.Response == nil || len(resp.Response.ScoreList) == 0 {
		return nil, fmt.Errorf("LKEAP rerank API returned empty score list")
	}

	scores := resp.Response.ScoreList
	if len(scores) != len(documents) {
		return nil, fmt.Errorf("LKEAP rerank score count mismatch: got %d scores for %d documents", len(scores), len(documents))
	}

	out := make([]RerankResult, 0, len(documents))
	for i, score := range scores {
		if score == nil {
			continue
		}
		out = append(out, RerankResult{Index: i, Score: *score})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Score > out[j].Score
	})
	return limitResults(out, topN), nil
}
