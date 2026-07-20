package phone

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
)

//go:embed phone.dat
var phoneDat []byte

const (
	carrierCMCC   byte = iota + 0x01 // 中国移动
	carrierCUCC                      // 中国联通
	carrierCTCC                      // 中国电信
	carrierCTCC_v                    // 电信虚拟运营商
	carrierCUCC_v                    // 联通虚拟运营商
	carrierCMCC_v                    // 移动虚拟运营商
	carrierCBCC                      // 中国广电
	carrierCBCC_v                    // 广电虚拟运营商

	intLen            = 4
	charLen           = 1
	phoneIndexLength  = 9
)

var carrierNames = map[byte]string{
	carrierCMCC:   "中国移动",
	carrierCUCC:   "中国联通",
	carrierCTCC:   "中国电信",
	carrierCBCC:   "中国广电",
	carrierCTCC_v: "中国电信虚拟运营商",
	carrierCUCC_v: "中国联通虚拟运营商",
	carrierCMCC_v: "中国移动虚拟运营商",
	carrierCBCC_v: "中国广电虚拟运营商",
}

// PhoneRecord holds mainland segment lookup metadata.
type PhoneRecord struct {
	PhoneNum string
	Province string
	City     string
	ZipCode  string
	AreaZone string
	CardType string
}

var (
	phoneContent     []byte
	phoneTotalLen    int32
	phoneFirstOffset int32
)

func init() {
	phoneContent = phoneDat
	phoneTotalLen = int32(len(phoneContent))
	phoneFirstOffset = readInt32(phoneContent[intLen : intLen*2])
}

func (pr PhoneRecord) String() string {
	return fmt.Sprintf(
		"PhoneNum: %s\nAreaZone: %s\nCardType: %s\nCity: %s\nZipCode: %s\nProvince: %s\n",
		pr.PhoneNum, pr.AreaZone, pr.CardType, pr.City, pr.ZipCode, pr.Province,
	)
}

func readInt32(b []byte) int32 {
	if len(b) < 4 {
		return 0
	}
	return int32(b[0]) | int32(b[1])<<8 | int32(b[2])<<16 | int32(b[3])<<24
}

func parsePhonePrefix(s string) (uint32, error) {
	var n, cutoff, maxVal uint32
	base := 10
	cutoff = (1<<32-1)/10 + 1
	maxVal = 1<<uint(32) - 1
	for i := 0; i < len(s); i++ {
		d := s[i]
		var v byte
		switch {
		case '0' <= d && d <= '9':
			v = d - '0'
		case 'a' <= d && d <= 'z':
			v = d - 'a' + 10
		case 'A' <= d && d <= 'Z':
			v = d - 'A' + 10
		default:
			return 0, errors.New("invalid syntax")
		}
		if v >= byte(base) {
			return 0, errors.New("invalid syntax")
		}
		if n >= cutoff {
			return 0, errors.New("value out of range")
		}
		n *= uint32(base)
		n1 := n + uint32(v)
		if n1 < n || n1 > maxVal {
			return 0, errors.New("value out of range")
		}
		n = n1
	}
	return n, nil
}

func phoneTotalRecord() int32 {
	return (int32(len(phoneContent)) - phoneFirstOffset) / phoneIndexLength
}

// Find looks up mainland mobile/landline segment data by phone number.
func Find(phoneNum string) (*PhoneRecord, error) {
	if len(phoneNum) < 7 || len(phoneNum) > 11 {
		return nil, errors.New("illegal phone length")
	}

	phoneSeven, err := parsePhonePrefix(phoneNum[:7])
	if err != nil {
		return nil, errors.New("illegal phone number")
	}
	phoneSevenInt32 := int32(phoneSeven)

	left := int32(0)
	right := (phoneTotalLen - phoneFirstOffset) / phoneIndexLength
	for left <= right {
		mid := (left + right) / 2
		offset := phoneFirstOffset + mid*phoneIndexLength
		if offset >= phoneTotalLen {
			break
		}
		curPhone := readInt32(phoneContent[offset : offset+intLen])
		recordOffset := readInt32(phoneContent[offset+intLen : offset+intLen*2])
		cardType := phoneContent[offset+intLen*2 : offset+intLen*2+charLen][0]
		switch {
		case curPhone > phoneSevenInt32:
			right = mid - 1
		case curPhone < phoneSevenInt32:
			left = mid + 1
		default:
			cbyte := phoneContent[recordOffset:]
			endOffset := int32(bytes.Index(cbyte, []byte("\000")))
			data := bytes.Split(cbyte[:endOffset], []byte("|"))
			cardStr, ok := carrierNames[cardType]
			if !ok {
				cardStr = "未知电信运营商"
			}
			return &PhoneRecord{
				PhoneNum: phoneNum,
				Province: string(data[0]),
				City:     string(data[1]),
				ZipCode:  string(data[2]),
				AreaZone: string(data[3]),
				CardType: cardStr,
			}, nil
		}
	}
	return nil, errors.New("phone's data not found")
}
