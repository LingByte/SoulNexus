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

func TestLocalStoreWriteOverwrite(t *testing.T) {
	store := NewLocalStore().(*LocalStore)
	store.Root = filepath.Join(t.TempDir(), "overwrite")
	key := "avatars/t1/u1.jpg"
	assert.NoError(t, store.Write(key, bytes.NewReader([]byte("first"))))
	assert.NoError(t, store.Write(key, bytes.NewReader([]byte("second"))))
	r, size, err := store.Read(key)
	assert.NoError(t, err)
	assert.Equal(t, int64(6), size)
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(r)
	assert.NoError(t, err)
	assert.Equal(t, "second", buf.String())
	_ = r.Close()
}

func TestLocalStoreWriteReplacesMistakenDirectory(t *testing.T) {
	store := NewLocalStore().(*LocalStore)
	store.Root = t.TempDir()
	key := "avatars/t1/u1.jpg"
	full := filepath.Join(store.Root, key)
	assert.NoError(t, os.MkdirAll(full, 0755))
	assert.NoError(t, os.WriteFile(filepath.Join(full, "nested"), []byte("x"), 0644))
	assert.NoError(t, store.Write(key, bytes.NewReader([]byte("avatar"))))
	ok, err := store.Exists(key)
	assert.NoError(t, err)
	assert.True(t, ok)
}
