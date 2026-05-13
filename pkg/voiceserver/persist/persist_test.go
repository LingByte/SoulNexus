// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package persist

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// newTestDB returns a fresh in-memory SQLite DB with both SIP entities
// migrated. Each test gets a private database so they cannot collide.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestExtractSIPUserPart(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`"Bob" <sip:13800138000@host.example.com>;tag=xyz`, "13800138000"},
		{`<sip:alice@example.com>`, "alice"},
		{`sip:bob@host`, "bob"},
		{`<sips:carol@host>;transport=tls`, "carol"},
		{`tel:+8613800138000`, "+8613800138000"},
		{`  `, ""},
	}
	for _, tc := range cases {
		if got := ExtractSIPUserPart(tc.in); got != tc.want {
			t.Errorf("ExtractSIPUserPart(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSIPCall_CreateAndFind(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	row := &SIPCall{
		CallID:     "abc-123",
		Direction:  DirectionInbound,
		FromHeader: `"Alice" <sip:alice@host>;tag=A1`,
		ToHeader:   `<sip:bob@host>`,
		FromNumber: ExtractSIPUserPart(`"Alice" <sip:alice@host>`),
		ToNumber:   "bob",
		State:      SIPCallStateRinging,
		InviteAt:   &now,
	}
	if err := CreateSIPCall(ctx, db, row); err != nil {
		t.Fatalf("create: %v", err)
	}
	if row.ID == 0 {
		t.Fatalf("expected auto-assigned ID")
	}
	got, err := FindSIPCallByCallID(ctx, db, "abc-123")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.CallID != row.CallID {
		t.Errorf("CallID = %q, want %q", got.CallID, row.CallID)
	}
	if got.State != SIPCallStateRinging {
		t.Errorf("State = %q, want %q", got.State, SIPCallStateRinging)
	}
}

func TestSIPCall_NotFoundIsTyped(t *testing.T) {
	db := newTestDB(t)
	_, err := FindSIPCallByCallID(context.Background(), db, "missing")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestSIPCall_AppendTurnCreatesRow(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	turn := SIPCallDialogTurn{
		ASRText: "你好",
		LLMText: "您好，请问有什么可以帮您？",
		At:      time.Now().UTC().Truncate(time.Millisecond),
	}
	n, err := AppendSIPCallTurn(ctx, db, "new-call", turn)
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if n != 1 {
		t.Fatalf("turn count = %d, want 1", n)
	}
	got, err := FindSIPCallByCallID(ctx, db, "new-call")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.State != SIPCallStateEstablished {
		t.Errorf("auto-created state = %q, want %q", got.State, SIPCallStateEstablished)
	}
	turns, err := UnmarshalSIPCallTurns(got.Turns)
	if err != nil {
		t.Fatalf("unmarshal turns: %v", err)
	}
	if len(turns) != 1 || turns[0].ASRText != "你好" {
		t.Errorf("turns roundtrip mismatch: %+v", turns)
	}
}

func TestSIPCall_AppendTurnGrowsExisting(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := AppendSIPCallTurn(ctx, db, "grow", SIPCallDialogTurn{
			ASRText: "u-" + string(rune('a'+i)),
			LLMText: "a-" + string(rune('a'+i)),
		})
		if err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}
	got, err := FindSIPCallByCallID(ctx, db, "grow")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.TurnCount != 3 {
		t.Errorf("TurnCount = %d, want 3", got.TurnCount)
	}
	turns, _ := UnmarshalSIPCallTurns(got.Turns)
	if len(turns) != 3 {
		t.Fatalf("len(turns) = %d, want 3", len(turns))
	}
	if got.FirstTurnAt == nil || got.LastTurnAt == nil {
		t.Errorf("expected FirstTurnAt and LastTurnAt to be set")
	}
}

func TestSIPCall_UpdateState(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	if err := CreateSIPCall(ctx, db, &SIPCall{CallID: "u1", State: SIPCallStateRinging}); err != nil {
		t.Fatalf("create: %v", err)
	}
	now := time.Now().UTC()
	n, err := UpdateSIPCallStateByCallID(ctx, db, "u1", map[string]any{
		"state":  SIPCallStateEstablished,
		"ack_at": now,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if n != 1 {
		t.Fatalf("rows-affected = %d, want 1", n)
	}
	got, _ := FindSIPCallByCallID(ctx, db, "u1")
	if got.State != SIPCallStateEstablished {
		t.Errorf("State = %q, want %q", got.State, SIPCallStateEstablished)
	}
	if got.AckAt == nil {
		t.Errorf("AckAt not set")
	}
}

func TestSIPCall_ListPage(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_ = CreateSIPCall(ctx, db, &SIPCall{
			CallID: "list-" + string(rune('A'+i)),
			State:  SIPCallStateEnded,
		})
	}
	list, total, err := ListSIPCallsPage(ctx, db, 1, 3, "", SIPCallStateEnded)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(list) != 3 {
		t.Errorf("page size = %d, want 3", len(list))
	}
}

func TestSIPUser_UpsertAndFind(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	expires := time.Now().Add(60 * time.Second).UTC()
	row := &SIPUser{
		Username:   "alice",
		Domain:     "example.com",
		ContactURI: "sip:alice@10.0.0.5:5060",
		RemoteIP:   "10.0.0.5",
		RemotePort: 5060,
		UserAgent:  "linphone/5.2",
		Online:     true,
		ExpiresAt:  &expires,
	}
	if err := UpsertSIPUser(ctx, db, row); err != nil {
		t.Fatalf("upsert insert: %v", err)
	}
	got, err := FindOnlineSIPUserByAOR(ctx, db, "alice", "example.com")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.ContactURI != row.ContactURI {
		t.Errorf("ContactURI = %q, want %q", got.ContactURI, row.ContactURI)
	}

	// Re-upsert with different contact: should update in place, not create.
	row2 := &SIPUser{
		Username:   "alice",
		Domain:     "example.com",
		ContactURI: "sip:alice@10.0.0.6:5060",
		RemoteIP:   "10.0.0.6",
		RemotePort: 5060,
		Online:     true,
		ExpiresAt:  &expires,
	}
	if err := UpsertSIPUser(ctx, db, row2); err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	got2, err := FindOnlineSIPUserByAOR(ctx, db, "alice", "example.com")
	if err != nil {
		t.Fatalf("find2: %v", err)
	}
	if got2.ContactURI != row2.ContactURI {
		t.Errorf("after update ContactURI = %q, want %q", got2.ContactURI, row2.ContactURI)
	}
	// Same row id (upsert, not insert).
	if got2.ID != got.ID {
		t.Errorf("id changed across upsert: %d → %d", got.ID, got2.ID)
	}
}

func TestSIPUser_OfflineFiltering(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// Online but expired in the past — should not be returned by FindOnline.
	past := time.Now().Add(-1 * time.Minute).UTC()
	if err := UpsertSIPUser(ctx, db, &SIPUser{
		Username: "expired", Domain: "x", Online: true, ExpiresAt: &past,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if _, err := FindOnlineSIPUserByAOR(ctx, db, "expired", "x"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected expired registration to be invisible, got %v", err)
	}
}

func TestSIPUser_SweepExpired(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	past := time.Now().Add(-1 * time.Minute).UTC()
	future := time.Now().Add(1 * time.Minute).UTC()
	_ = UpsertSIPUser(ctx, db, &SIPUser{Username: "u1", Domain: "x", Online: true, ExpiresAt: &past})
	_ = UpsertSIPUser(ctx, db, &SIPUser{Username: "u2", Domain: "x", Online: true, ExpiresAt: &future})

	n, err := SweepExpiredSIPUsers(ctx, db)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if n != 1 {
		t.Fatalf("swept = %d, want 1", n)
	}
	list, total, err := ListSIPUsersPage(ctx, db, 1, 50, true)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].Username != "u2" {
		t.Errorf("after sweep online list = %+v (total=%d)", list, total)
	}
}
