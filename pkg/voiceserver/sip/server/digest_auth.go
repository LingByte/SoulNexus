package server

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/uas"
)

type digestNonce struct {
	expires time.Time
}

type sipDigestAuth struct {
	realm string
	user  string
	pass  string
	mu     sync.Mutex
	nonces map[string]digestNonce
}

func newSIPDigest(realm, user, pass string) *sipDigestAuth {
	realm = strings.TrimSpace(realm)
	user = strings.TrimSpace(user)
	pass = strings.TrimSpace(pass)
	if realm == "" || user == "" || pass == "" {
		return nil
	}
	return &sipDigestAuth{
		realm:  realm,
		user:   user,
		pass:   pass,
		nonces: make(map[string]digestNonce),
	}
}

func (d *sipDigestAuth) gc() {
	if d == nil {
		return
	}
	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()
	for k, v := range d.nonces {
		if now.After(v.expires) {
			delete(d.nonces, k)
		}
	}
}

func (d *sipDigestAuth) issueNonce() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	}
	n := hex.EncodeToString(b[:])
	d.mu.Lock()
	d.nonces[n] = digestNonce{expires: time.Now().Add(10 * time.Minute)}
	d.mu.Unlock()
	return n
}

func (d *sipDigestAuth) challenge401(req *stack.Message) (*stack.Message, error) {
	if d == nil || req == nil {
		return nil, fmt.Errorf("sip/server: digest")
	}
	d.gc()
	nonce := d.issueNonce()
	www := fmt.Sprintf(`Digest realm="%s", nonce="%s", algorithm=MD5, qop="auth"`, d.realm, nonce)
	resp, err := uas.NewResponse(req, 401, "Unauthorized", "", "")
	if err != nil {
		return nil, err
	}
	resp.SetHeader("WWW-Authenticate", www)
	return resp, nil
}

func md5hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

func digestHA1(user, realm, pass string) string {
	return md5hex(fmt.Sprintf("%s:%s:%s", user, realm, pass))
}

func digestExpectResponse(auth map[string]string, method, uri string, ha1 string) string {
	nonce := auth["nonce"]
	qop := strings.Trim(auth["qop"], `"`)
	if qop == "auth" {
		nc := auth["nc"]
		cnonce := strings.Trim(auth["cnonce"], `"`)
		ha2 := md5hex(fmt.Sprintf("%s:%s", method, uri))
		return md5hex(fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, qop, ha2))
	}
	ha2 := md5hex(fmt.Sprintf("%s:%s", method, uri))
	return md5hex(fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2))
}

func parseDigestAuth(h string) map[string]string {
	out := make(map[string]string)
	h = strings.TrimPrefix(strings.TrimSpace(h), "Digest")
	for _, part := range strings.Split(h, ",") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(strings.ToLower(kv[0]))
		v := strings.Trim(strings.TrimSpace(kv[1]), `"`)
		out[k] = v
	}
	return out
}

func (d *sipDigestAuth) verifyINVITE(req *stack.Message) bool {
	if d == nil || req == nil {
		return false
	}
	raw := req.GetHeader("Authorization")
	if raw == "" {
		raw = req.GetHeader("Proxy-Authorization")
	}
	if raw == "" || !strings.HasPrefix(strings.TrimSpace(strings.ToLower(raw)), "digest") {
		return false
	}
	auth := parseDigestAuth(raw)
	if strings.TrimSpace(strings.ToLower(auth["username"])) != strings.ToLower(d.user) ||
		strings.TrimSpace(strings.ToLower(auth["realm"])) != strings.ToLower(d.realm) {
		return false
	}
	nonce := auth["nonce"]
	d.mu.Lock()
	_, ok := d.nonces[nonce]
	if ok {
		delete(d.nonces, nonce)
	}
	d.mu.Unlock()
	if !ok {
		return false
	}
	uri := auth["uri"]
	if uri == "" {
		uri = req.RequestURI
	}
	response := strings.Trim(auth["response"], `"`)
	if response == "" {
		return false
	}
	ha1 := digestHA1(d.user, d.realm, d.pass)
	expect := digestExpectResponse(auth, req.Method, uri, ha1)
	return strings.EqualFold(expect, response)
}
