// utils_secure_query_test.go
package utils

import (
	"fmt"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glog "gorm.io/gorm/logger"
)

// ---------- 测试模型 ----------
type TestModel struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:100"`
	Email     string `gorm:"size:100"`
	Age       int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ---------- 测试 DB 初始化（静音 + 忽略 NotFound） ----------
func setupTestDB(t *testing.T) *gorm.DB {
	silentLogger := glog.New(
		log.New(io.Discard, "", log.LstdFlags),
		glog.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  glog.Silent,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: silentLogger})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&TestModel{}))
	return db
}

// ---------- 基础数据 ----------
func seedUsers(t *testing.T, db *gorm.DB, n int) {
	for i := 0; i < n; i++ {
		m := &TestModel{
			Name:  fmt.Sprintf("User %d", i),
			Email: fmt.Sprintf("u%d@example.com", i),
			Age:   18 + i,
		}
		require.NoError(t, db.Create(m).Error)
	}
}

// ---------- SafeWhere 覆盖更多操作符 ----------
func TestSecureQueryBuilder_SafeWhere_AllOps(t *testing.T) {
	db := setupTestDB(t)
	b := NewSecureQueryBuilder(db)
	seedUsers(t, db, 5) // Age: 18..22

	t.Run("equals", func(t *testing.T) {
		var got []TestModel
		q := b.SafeWhere("age", "=", 20)
		require.NoError(t, q.Find(&got).Error)
		assert.Len(t, got, 1)
	})

	t.Run("not equals", func(t *testing.T) {
		var got []TestModel
		q := b.SafeWhere("age", "!=", 20)
		require.NoError(t, q.Find(&got).Error)
		assert.Len(t, got, 4)
	})

	t.Run("<>", func(t *testing.T) {
		var got []TestModel
		q := b.SafeWhere("age", "<>", 20)
		require.NoError(t, q.Find(&got).Error)
		assert.Len(t, got, 4)
	})

	t.Run("gt/lt/gte/lte", func(t *testing.T) {
		var a, b1, c, d []TestModel
		require.NoError(t, b.SafeWhere("age", ">", 20).Find(&a).Error)
		require.NoError(t, b.SafeWhere("age", "<", 20).Find(&b1).Error)
		require.NoError(t, b.SafeWhere("age", ">=", 20).Find(&c).Error)
		require.NoError(t, b.SafeWhere("age", "<=", 20).Find(&d).Error)
		assert.Equal(t, 2, len(a))  // 21,22
		assert.Equal(t, 2, len(b1)) // 18,19
		assert.Equal(t, 3, len(c))  // 20,21,22
		assert.Equal(t, 3, len(d))  // 18,19,20
	})

	t.Run("like string", func(t *testing.T) {
		var got []TestModel
		q := b.SafeWhere("email", "LIKE", "u1@")
		require.NoError(t, q.Find(&got).Error)
		assert.Len(t, got, 1)
	})

	t.Run("not like string", func(t *testing.T) {
		var got []TestModel
		q := b.SafeWhere("email", "NOT LIKE", "u1@")
		require.NoError(t, q.Find(&got).Error)
		assert.Len(t, got, 4)
	})

	t.Run("like non-string (still param)", func(t *testing.T) {
		var got []TestModel
		q := b.SafeWhere("email", "LIKE", 123)
		require.NoError(t, q.Find(&got).Error)
		assert.Len(t, got, 0)
	})

	t.Run("IN ints", func(t *testing.T) {
		var got []TestModel
		q := b.SafeWhere("age", "IN", []int{18, 22})
		require.NoError(t, q.Find(&got).Error)
		assert.Len(t, got, 2)
	})

	t.Run("NOT IN strings", func(t *testing.T) {
		var got []TestModel
		q := b.SafeWhere("name", "NOT IN", []string{"User 0", "User 1"})
		require.NoError(t, q.Find(&got).Error)
		assert.Len(t, got, 3)
	})

	t.Run("BETWEEN ok", func(t *testing.T) {
		var got []TestModel
		q := b.SafeWhere("age", "BETWEEN", []interface{}{19, 21})
		require.NoError(t, q.Find(&got).Error)
		assert.Len(t, got, 3) // 19,20,21
	})

	t.Run("BETWEEN invalid slice len -> panic", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = b.SafeWhere("age", "BETWEEN", []interface{}{19})
		})
	})

	t.Run("IS NULL / IS NOT NULL", func(t *testing.T) {
		var got []TestModel
		require.NoError(t, b.SafeWhere("email", "IS NULL", nil).Find(&got).Error)
		assert.Len(t, got, 0)
		require.NoError(t, b.SafeWhere("email", "IS NOT NULL", nil).Find(&got).Error)
		assert.Len(t, got, 5)
	})

	t.Run("invalid column should panic", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = b.SafeWhere("age; DROP TABLE users;", "=", 1)
		})
	})
}

// ---------- SafeOrder/SafeSelect/SafeGroup ----------
func TestSecureQueryBuilder_OrderSelectGroup(t *testing.T) {
	db := setupTestDB(t)
	b := NewSecureQueryBuilder(db)
	seedUsers(t, db, 3)

	assert.NotPanics(t, func() { _ = b.SafeOrder("age", "ASC") })
	assert.NotPanics(t, func() { _ = b.SafeOrder("age", "DESC") })
	assert.NotPanics(t, func() { _ = b.SafeOrder("age", "INVALID") }) // 默认升序
	assert.Panics(t, func() { _ = b.SafeOrder("age desc; DROP", "ASC") })

	assert.NotPanics(t, func() { _ = b.SafeSelect([]string{"name", "email"}) })
	assert.Panics(t, func() { _ = b.SafeSelect([]string{"name", "email; DROP"}) })

	assert.NotPanics(t, func() { _ = b.SafeGroup([]string{"name", "email"}) })
	assert.Panics(t, func() { _ = b.SafeGroup([]string{"name", "email; DROP"}) })
}

// ---------- SafeHaving / SafeQuery ----------
func TestSecureQueryBuilder_HavingAndQuery(t *testing.T) {
	db := setupTestDB(t)
	b := NewSecureQueryBuilder(db)
	seedUsers(t, db, 5)

	t.Run("having valid with args", func(t *testing.T) {
		type Row struct{ C int64 }
		var rows []Row

		q := b.
			SafeHaving("COUNT(*) > ?", 2). // 先加 Having（来自 b.Db）
			Model(&TestModel{}).           // 再补 Model
			Select("COUNT(*) AS c")        // 再补 Select

		err := q.Scan(&rows).Error
		require.NoError(t, err)
		require.NotEmpty(t, rows)
		assert.Greater(t, rows[0].C, int64(0))
	})

	t.Run("having invalid - union", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = b.SafeHaving("COUNT(*) > 0 UNION SELECT * FROM users")
		})
	})

	t.Run("raw select ok", func(t *testing.T) {
		q := b.SafeQuery("SELECT name FROM test_models WHERE age > ?", 20)
		var rows []map[string]any
		require.NoError(t, q.Find(&rows).Error)
	})

	t.Run("raw forbidden - drop", func(t *testing.T) {
		assert.Panics(t, func() { _ = b.SafeQuery("DROP TABLE test_models") })
	})

	t.Run("raw forbidden - not start with select", func(t *testing.T) {
		assert.Panics(t, func() { _ = b.SafeQuery("UPDATE test_models SET name='x'") })
	})
}

// ---------- Paginate / Count / Exists / First / Find ----------
func TestSecureQueryBuilder_PaginateAndCrud(t *testing.T) {
	db := setupTestDB(t)
	b := NewSecureQueryBuilder(db)
	seedUsers(t, db, 25)

	t.Run("paginate", func(t *testing.T) {
		page1 := b.SafePaginate(1, 10)
		page2 := b.SafePaginate(2, 10)
		page3 := b.SafePaginate(3, 10) // only 5 left

		var a, b1, c []TestModel
		require.NoError(t, page1.Find(&a).Error)
		require.NoError(t, page2.Find(&b1).Error)
		require.NoError(t, page3.Find(&c).Error)
		assert.Equal(t, 10, len(a))
		assert.Equal(t, 10, len(b1))
		assert.Equal(t, 5, len(c))

		var d []TestModel
		require.NoError(t, b.SafePaginate(0, 0).Find(&d).Error)
		assert.Equal(t, 20, len(d)) // 默认 20
	})

	t.Run("count", func(t *testing.T) {
		cnt, err := b.SafeCount(&TestModel{})
		require.NoError(t, err)
		assert.Equal(t, int64(25), cnt)
	})

	t.Run("exists", func(t *testing.T) {
		ok, err := b.SafeExists(&TestModel{}, map[string]any{"name": "User 1"})
		require.NoError(t, err)
		assert.True(t, ok)

		ok, err = b.SafeExists(&TestModel{}, map[string]any{"name": "NotFound"})
		require.NoError(t, err)
		assert.False(t, ok)

		_, err = b.SafeExists(&TestModel{}, map[string]any{"name; DROP": "x"})
		assert.Error(t, err)
	})

	t.Run("first", func(t *testing.T) {
		var m TestModel
		err := b.SafeFirst(&m, map[string]any{"name": "User 2"})
		assert.NoError(t, err)
		assert.Equal(t, "User 2", m.Name)

		err = b.SafeFirst(&m, map[string]any{"name": "NotFound"})
		assert.Error(t, err)

		err = b.SafeFirst(&m, map[string]any{"name; DROP": "x"})
		assert.Error(t, err)
	})

	t.Run("find", func(t *testing.T) {
		var ms []TestModel
		err := b.SafeFind(&ms, map[string]any{"name": "User 2"})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(ms))

		err = b.SafeFind(&ms, map[string]any{"name": "NotFound"})
		assert.NoError(t, err)
		assert.Equal(t, 0, len(ms))

		err = b.SafeFind(&ms, map[string]any{"name; DROP": "x"})
		assert.Error(t, err)
	})
}

// ---------- SanitizeValue / ValidateInput 追加覆盖 ----------
func TestSanitizeValue_MoreCases(t *testing.T) {
	type S struct {
		A string
		B int
	}
	res := SanitizeValue(S{A: "x'y", B: 3})
	_, ok := res.(string)
	assert.True(t, ok)
	assert.True(t, strings.Contains(res.(string), "x''y"))
}

func TestValidateInput_MorePatterns(t *testing.T) {
	bads := []string{
		"exec(",            // exec(
		"INSERT INTO t",    // insert into <tbl>
		"UPDATE x SET y=1", // update <tbl> set
		"DROP TABLE t",     // drop
		"or 1 = 1",         // or 1=1
		"UNION SELECT *",   // union select
	}
	for _, s := range bads {
		assert.Error(t, ValidateInput(s), s)
	}
	assert.NoError(t, ValidateInput("普通安全字符串"))
	assert.NoError(t, ValidateInput(12345))
}
