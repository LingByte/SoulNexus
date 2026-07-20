package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/LingByte/SoulNexus/pkg/utils"
)

func knowledgeDocRawKey(groupID uint, namespace string, docID uint, fileName string) string {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = "default"
	}
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(fileName)))
	if ext == "" {
		ext = ".bin"
	}
	return path.Join("knowledge", fmt.Sprintf("%d", groupID), ns, fmt.Sprintf("%d%s", docID, ext))
}

func knowledgeDocTextKey(groupID uint, namespace string, docID uint) string {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = "default"
	}
	return path.Join("knowledge", fmt.Sprintf("%d", groupID), ns, fmt.Sprintf("%d.txt", docID))
}

func writeKnowledgeDocBytes(key string, body []byte) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("storage key is required")
	}
	return stores.Default().Write(key, bytes.NewReader(body))
}

func writeKnowledgeDocContent(key, content string) error {
	return writeKnowledgeDocBytes(key, []byte(content))
}

func readKnowledgeDocBytes(key string) ([]byte, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("storage key is required")
	}
	rc, _, err := stores.Default().Read(key)
	if err != nil {
		if errors.Is(err, utils.ErrAttachmentNotExist) {
			return nil, fmt.Errorf("%w: %s", utils.ErrAttachmentNotExist, key)
		}
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func readKnowledgeDocContent(key string) (string, error) {
	body, err := readKnowledgeDocBytes(key)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func deleteKnowledgeDocContent(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	st := stores.Default()
	ok, err := st.Exists(key)
	if err != nil || !ok {
		return err
	}
	return st.Delete(key)
}

func deleteKnowledgeDocFiles(rawKey, textKey string) {
	_ = deleteKnowledgeDocContent(rawKey)
	_ = deleteKnowledgeDocContent(textKey)
}
