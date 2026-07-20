// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package wxcrypt implements WeChat/WeCom callback message encryption (AES-CBC).
package wxcrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Crypt handles token + EncodingAESKey verify/decrypt/encrypt.
type Crypt struct {
	Token          string
	EncodingAESKey string
	AppID          string // corpId or official account appId
	aesKey         []byte
}

// New builds a Crypt from token and base64 EncodingAESKey (43 chars + =).
func New(token, encodingAESKey, appID string) (*Crypt, error) {
	token = strings.TrimSpace(token)
	encodingAESKey = strings.TrimSpace(encodingAESKey)
	appID = strings.TrimSpace(appID)
	if token == "" || encodingAESKey == "" {
		return nil, errors.New("wxcrypt: token and encodingAESKey required")
	}
	key, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil || len(key) != 32 {
		return nil, errors.New("wxcrypt: invalid encodingAESKey")
	}
	return &Crypt{Token: token, EncodingAESKey: encodingAESKey, AppID: appID, aesKey: key}, nil
}

// VerifyURL validates GET echostr for callback URL configuration.
func (c *Crypt) VerifyURL(signature, timestamp, nonce, echostr string) (string, error) {
	if !c.checkSignature(signature, timestamp, nonce, echostr) {
		return "", errors.New("wxcrypt: signature mismatch")
	}
	plain, err := c.decrypt(echostr)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// DecryptMsg decrypts a POST callback body (XML with Encrypt field).
func (c *Crypt) DecryptMsg(signature, timestamp, nonce string, body []byte) ([]byte, error) {
	var envelope struct {
		ToUserName string `xml:"ToUserName"`
		Encrypt    string `xml:"Encrypt"`
		AgentID    string `xml:"AgentID"`
	}
	if err := xml.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if envelope.Encrypt == "" {
		return body, nil // plaintext mode
	}
	if !c.checkSignature(signature, timestamp, nonce, envelope.Encrypt) {
		return nil, errors.New("wxcrypt: msg signature mismatch")
	}
	return c.decrypt(envelope.Encrypt)
}

// EncryptMsg wraps plaintext XML into an encrypted response envelope.
func (c *Crypt) EncryptMsg(reply []byte, timestamp, nonce string) ([]byte, error) {
	cipherText, err := c.encrypt(reply)
	if err != nil {
		return nil, err
	}
	sig := c.sign(timestamp, nonce, cipherText)
	type xmlResp struct {
		XMLName   xml.Name `xml:"xml"`
		Encrypt   CDATA    `xml:"Encrypt"`
		MsgSig    CDATA    `xml:"MsgSignature"`
		TimeStamp string   `xml:"TimeStamp"`
		Nonce     CDATA    `xml:"Nonce"`
	}
	out, err := xml.Marshal(xmlResp{
		Encrypt:   CDATA(cipherText),
		MsgSig:    CDATA(sig),
		TimeStamp: timestamp,
		Nonce:     CDATA(nonce),
	})
	return out, err
}

// CDATA marshals as XML CDATA section.
type CDATA string

func (c CDATA) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(struct {
		string `xml:",cdata"`
	}{string(c)}, start)
}

func (c *Crypt) checkSignature(signature, timestamp, nonce, encrypt string) bool {
	return strings.EqualFold(c.sign(timestamp, nonce, encrypt), signature)
}

func (c *Crypt) sign(timestamp, nonce, encrypt string) string {
	arr := []string{c.Token, timestamp, nonce, encrypt}
	sort.Strings(arr)
	h := sha1.New()
	_, _ = io.WriteString(h, strings.Join(arr, ""))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (c *Crypt) decrypt(cipherB64 string) ([]byte, error) {
	cipherData, err := base64.StdEncoding.DecodeString(cipherB64)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(c.aesKey)
	if err != nil {
		return nil, err
	}
	if len(cipherData) < aes.BlockSize || len(cipherData)%aes.BlockSize != 0 {
		return nil, errors.New("wxcrypt: bad ciphertext length")
	}
	iv := c.aesKey[:16]
	mode := cipher.NewCBCDecrypter(block, iv)
	plain := make([]byte, len(cipherData))
	mode.CryptBlocks(plain, cipherData)
	plain, err = pkcs7Unpad(plain)
	if err != nil {
		return nil, err
	}
	if len(plain) < 20 {
		return nil, errors.New("wxcrypt: plain too short")
	}
	msgLen := binary.BigEndian.Uint32(plain[16:20])
	if int(20+msgLen) > len(plain) {
		return nil, errors.New("wxcrypt: invalid msg length")
	}
	msg := plain[20 : 20+msgLen]
	receiveID := string(plain[20+msgLen:])
	if c.AppID != "" && receiveID != "" && receiveID != c.AppID {
		return nil, fmt.Errorf("wxcrypt: appid mismatch want %s got %s", c.AppID, receiveID)
	}
	return msg, nil
}

func (c *Crypt) encrypt(msg []byte) (string, error) {
	randBytes := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, randBytes); err != nil {
		return "", err
	}
	buf := bytes.NewBuffer(randBytes)
	_ = binary.Write(buf, binary.BigEndian, uint32(len(msg)))
	buf.Write(msg)
	buf.WriteString(c.AppID)
	plain := pkcs7Pad(buf.Bytes(), aes.BlockSize)
	block, err := aes.NewCipher(c.aesKey)
	if err != nil {
		return "", err
	}
	iv := c.aesKey[:16]
	mode := cipher.NewCBCEncrypter(block, iv)
	cipherData := make([]byte, len(plain))
	mode.CryptBlocks(cipherData, plain)
	return base64.StdEncoding.EncodeToString(cipherData), nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	pad := blockSize - len(data)%blockSize
	if pad == 0 {
		pad = blockSize
	}
	return append(data, bytes.Repeat([]byte{byte(pad)}, pad)...)
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("wxcrypt: empty")
	}
	pad := int(data[len(data)-1])
	if pad < 1 || pad > len(data) {
		return nil, errors.New("wxcrypt: bad padding")
	}
	return data[:len(data)-pad], nil
}
