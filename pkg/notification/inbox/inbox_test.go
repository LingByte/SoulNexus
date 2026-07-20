package inbox

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openInboxDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&Message{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestService_Send(t *testing.T) {
	t.Parallel()
	db := openInboxDB(t)
	svc := NewService(db)
	if err := svc.Send(10, "title", "content"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	var n int64
	db.Model(&Message{}).Where("user_id = ?", 10).Count(&n)
	if n != 1 {
		t.Errorf("count = %d", n)
	}
}

func TestService_Send_nilService(t *testing.T) {
	t.Parallel()
	var svc *Service
	if err := svc.Send(1, "t", "c"); err != gorm.ErrInvalidDB {
		t.Errorf("err = %v", err)
	}
}

func TestService_UnreadCount(t *testing.T) {
	t.Parallel()
	db := openInboxDB(t)
	svc := NewService(db)
	_ = svc.Send(1, "a", "b")
	_ = svc.Send(1, "c", "d")
	count, err := svc.UnreadCount(1)
	if err != nil || count != 2 {
		t.Fatalf("UnreadCount = %d err %v", count, err)
	}
}

func TestService_MarkRead(t *testing.T) {
	t.Parallel()
	db := openInboxDB(t)
	svc := NewService(db)
	_ = svc.Send(1, "a", "b")
	var msg Message
	db.First(&msg)
	if err := svc.MarkRead(1, msg.ID); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	got, _ := svc.GetOne(1, msg.ID)
	if !got.Read {
		t.Error("expected read=true")
	}
}

func TestService_MarkAllRead(t *testing.T) {
	t.Parallel()
	db := openInboxDB(t)
	svc := NewService(db)
	_ = svc.Send(1, "a", "b")
	_ = svc.Send(1, "c", "d")
	if err := svc.MarkAllRead(1); err != nil {
		t.Fatalf("MarkAllRead: %v", err)
	}
	count, _ := svc.UnreadCount(1)
	if count != 0 {
		t.Errorf("unread = %d", count)
	}
}

func TestService_Delete(t *testing.T) {
	t.Parallel()
	db := openInboxDB(t)
	svc := NewService(db)
	_ = svc.Send(1, "a", "b")
	var msg Message
	db.First(&msg)
	if err := svc.Delete(1, msg.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := svc.GetOne(1, msg.ID)
	if err == nil {
		t.Error("expected not found after delete")
	}
}

func TestService_BatchDelete(t *testing.T) {
	t.Parallel()
	db := openInboxDB(t)
	svc := NewService(db)
	_ = svc.Send(1, "a", "b")
	_ = svc.Send(1, "c", "d")
	var ids []uint
	db.Model(&Message{}).Pluck("id", &ids)
	n, err := svc.BatchDelete(1, ids)
	if err != nil || n != 2 {
		t.Fatalf("BatchDelete = %d err %v", n, err)
	}
	n, _ = svc.BatchDelete(1, nil)
	if n != 0 {
		t.Errorf("empty batch = %d", n)
	}
}

func TestService_ListPage(t *testing.T) {
	t.Parallel()
	db := openInboxDB(t)
	svc := NewService(db)
	_ = svc.Send(1, "welcome", "hello world")
	_ = svc.Send(1, "other", "bye")
	res, err := svc.ListPage(1, 1, 10, "unread", "welcome", "", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("ListPage: %v", err)
	}
	if len(res.List) != 1 || res.List[0].Title != "welcome" {
		t.Errorf("list = %+v", res.List)
	}
	if res.TotalUnread < 1 {
		t.Errorf("TotalUnread = %d", res.TotalUnread)
	}
}

func TestService_AllIDs(t *testing.T) {
	t.Parallel()
	db := openInboxDB(t)
	svc := NewService(db)
	_ = svc.Send(2, "x", "y")
	ids, err := svc.AllIDs(2, "", "", "", time.Time{}, time.Time{})
	if err != nil || len(ids) != 1 {
		t.Fatalf("AllIDs = %v err %v", ids, err)
	}
}

func TestService_CleanOldUnread(t *testing.T) {
	t.Parallel()
	db := openInboxDB(t)
	svc := NewService(db)
	_ = svc.Send(1, "old", "msg")
	var msg Message
	db.First(&msg)
	db.Model(&msg).Update("created_at", time.Now().Add(-48*time.Hour))
	n, err := svc.CleanOldUnread(time.Now().Add(-24 * time.Hour))
	if err != nil || n != 1 {
		t.Fatalf("CleanOldUnread = %d err %v", n, err)
	}
}

func TestMessageTableName(t *testing.T) {
	t.Parallel()
	if (Message{}).TableName() != "internal_notifications" {
		t.Error("unexpected table name")
	}
}
