package sipcampaign

import (
	"testing"

	"github.com/LingByte/SoulNexus/internal/models"
)

func TestParseCorrelation(t *testing.T) {
	campaignID, contactID, attemptNo, ok := parseCorrelation("camp:12:contact:34:attempt:2")
	if !ok {
		t.Fatalf("expected ok")
	}
	if campaignID != 12 || contactID != 34 || attemptNo != 2 {
		t.Fatalf("unexpected parsed values: %d %d %d", campaignID, contactID, attemptNo)
	}
}

func TestBuildDialTarget_FromTemplate(t *testing.T) {
	c := models.SIPCampaign{
		RequestURIFmt: "sip:%s@10.0.0.8:5060",
		SignalingAddr: "10.0.0.8:5060",
	}
	ct := models.SIPCampaignContact{Phone: "1001"}
	target, err := buildDialTarget(c, ct)
	if err != nil {
		t.Fatalf("buildDialTarget err=%v", err)
	}
	if target.RequestURI != "sip:1001@10.0.0.8:5060" {
		t.Fatalf("unexpected request uri: %s", target.RequestURI)
	}
}

