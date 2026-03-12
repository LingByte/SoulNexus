package recognizer

import (
	"bytes"
	"encoding/binary"

	"github.com/bytedance/sonic"
)

type UserMeta struct {
	UID        string `json:"uid,omitempty"`
	DID        string `json:"did,omitempty"`
	Platform   string `json:"platform,omitempty"`
	SDKVersion string `json:"sdk_version,omitempty"`
	APPVersion string `json:"app_version,omitempty"`
}

type AudioMeta struct {
	Format  string `json:"format,omitempty"`
	Codec   string `json:"codec,omitempty"`
	Rate    int    `json:"rate,omitempty"`
	Bits    int    `json:"bits,omitempty"`
	Channel int    `json:"channel,omitempty"`
}

type CorpusMeta struct {
	BoostingTableName string `json:"boosting_table_name,omitempty"`
	CorrectTableName  string `json:"correct_table_name,omitempty"`
	Context           string `json:"context,omitempty"`
}

type RequestMeta struct {
	ModelName       string     `json:"model_name,omitempty"`
	EnableITN       bool       `json:"enable_itn,omitempty"`
	EnablePUNC      bool       `json:"enable_punc,omitempty"`
	EnableDDC       bool       `json:"enable_ddc,omitempty"`
	ShowUtterances  bool       `json:"show_utterances"`
	EnableNonstream bool       `json:"enable_nonstream"`
	Corpus          CorpusMeta `json:"corpus,omitempty"`
}

type RequestPayload struct {
	User    UserMeta    `json:"user"`
	Audio   AudioMeta   `json:"audio"`
	Request RequestMeta `json:"request"`
}

// NewFullClientRequest creates a full client request payload
func NewFullClientRequest(config *Config) []byte {
	var request bytes.Buffer
	request.Write(DefaultHeader().WithMessageTypeSpecificFlags(FlagPosSequence).toBytes())
	payload := RequestPayload{
		User: UserMeta{
			UID:        config.User.UID,
			DID:        config.User.DID,
			Platform:   config.User.Platform,
			SDKVersion: config.User.SDKVersion,
			APPVersion: config.User.APPVersion,
		},
		Audio: AudioMeta{
			Format:  config.Audio.Format,
			Codec:   config.Audio.Codec,
			Rate:    config.Audio.Rate,
			Bits:    config.Audio.Bits,
			Channel: config.Audio.Channel,
		},
		Request: RequestMeta{
			ModelName:       config.Request.ModelName,
			EnableITN:       config.Request.EnableITN,
			EnablePUNC:      config.Request.EnablePUNC,
			EnableDDC:       config.Request.EnableDDC,
			ShowUtterances:  config.Request.ShowUtterances,
			EnableNonstream: config.Request.EnableNonstream,
			Corpus: CorpusMeta{
				BoostingTableName: config.Request.Corpus.BoostingTableName,
				CorrectTableName:  config.Request.Corpus.CorrectTableName,
				Context:           config.Request.Corpus.Context,
			},
		},
	}
	payloadArr, _ := sonic.Marshal(payload)
	payloadArr = GzipCompress(payloadArr)
	payloadSize := len(payloadArr)
	payloadSizeArr := make([]byte, 4)
	binary.BigEndian.PutUint32(payloadSizeArr, uint32(payloadSize))
	_ = binary.Write(&request, binary.BigEndian, int32(1))
	request.Write(payloadSizeArr)
	request.Write(payloadArr)
	return request.Bytes()
}

func NewAudioOnlyRequest(seq int, segment []byte) []byte {
	var request bytes.Buffer
	header := DefaultHeader()
	if seq < 0 {
		header.WithMessageTypeSpecificFlags(FlagNegWithSequence)
	} else {
		header.WithMessageTypeSpecificFlags(FlagPosSequence)
	}
	header.WithMessageType(MessageTypeClientAudioOnlyRequest)
	request.Write(header.toBytes())

	// write seq
	_ = binary.Write(&request, binary.BigEndian, int32(seq))
	// write payload size
	payload := GzipCompress(segment)
	_ = binary.Write(&request, binary.BigEndian, int32(len(payload)))
	// write payload
	request.Write(payload)
	return request.Bytes()
}
