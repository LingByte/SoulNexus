package intentonnx

import (
	"errors"
	"strings"

	smtok "github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
)

var errPadToken = errors.New("intentonnx: could not resolve pad token from tokenizer")

func newPreparedTokenizer(path string, seqLen int) (*smtok.Tokenizer, error) {
	tk, err := pretrained.FromFile(strings.TrimSpace(path))
	if err != nil {
		return nil, err
	}
	padID, padTok, ok := findPadToken(tk)
	if !ok {
		return nil, errPadToken
	}
	_ = padID
	tk.WithTruncation(&smtok.TruncationParams{MaxLength: seqLen, Strategy: smtok.LongestFirst, Stride: 0})
	tk.WithPadding(&smtok.PaddingParams{
		Strategy:  *smtok.NewPaddingStrategy(smtok.WithFixed(seqLen)),
		Direction: smtok.Right,
		PadId:     padID,
		PadTypeId: 0,
		PadToken:  padTok,
	})
	return tk, nil
}

func findPadToken(tk *smtok.Tokenizer) (id int, tok string, ok bool) {
	pad := string([]byte{0x3c, 0x50, 0x41, 0x44, 0x3e})
	padBracket := string([]byte{0x5b, 0x50, 0x41, 0x44, 0x5d})
	for _, c := range []string{pad, strings.ToLower(pad), padBracket, strings.ToLower(padBracket), "<|pad|>"} {
		if i, o := tk.TokenToId(c); o {
			return i, c, true
		}
	}
	return 0, "", false
}
