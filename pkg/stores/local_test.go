package stores

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalStore(t *testing.T) {
	store := NewLocalStore().(*LocalStore)
	assert.NotNil(t, store)
	store.Root = filepath.Join(t.TempDir(), "unittest")
	os.RemoveAll(store.Root)
	fname := "test.txt"
	ok, err := store.Exists(fname)
	assert.NoError(t, err)
	assert.False(t, ok)

	err = store.Write(fname, bytes.NewReader([]byte("hello")))
	assert.NoError(t, err)

	ok, err = store.Exists(fname)
	assert.NoError(t, err)
	assert.True(t, ok)

	r, size, err := store.Read(fname)
	assert.Equal(t, int64(5), size)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(r)
	assert.NoError(t, err)
	assert.Equal(t, "hello", buf.String())
	r.Close()
	err = store.Delete(fname)
	assert.NoError(t, err)

	fullpath := store.PublicURL(fname)
	t.Log("fullpath:", fullpath)
	assert.True(t, strings.HasSuffix(fullpath, "test.txt"))

	ok, err = store.Exists(fname)
	assert.NoError(t, err)
	assert.False(t, ok)

	err = store.Delete("../../not_exist.txt")
	assert.EqualError(t, err, ErrInvalidPath.Error())
}
