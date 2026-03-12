package utils

import (
	"strconv"
	"strings"
)

func keyTag(key string) string {
	i := strings.IndexByte(key, '{')
	if i < 0 {
		return key
	}
	j := strings.IndexByte(key[i+1:], '}')
	if j < 0 {
		return key
	}
	if j == 0 { // "{}" empty tag
		return key
	}
	return key[i+1 : i+1+j]
}

func makeCRC16Table(poly uint16) [256]uint16 {
	var tab [256]uint16
	for i := 0; i < 256; i++ {
		var crc uint16 = uint16(i) << 8
		for j := 0; j < 8; j++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ poly
			} else {
				crc <<= 1
			}
		}
		tab[i] = crc
	}
	return tab
}

var crc16Tab = makeCRC16Table(0x1021)

func crc16CCITT(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		idx := byte((crc >> 8) ^ uint16(b)) // High byte XOR data
		crc = (crc << 8) ^ crc16Tab[idx]
	}
	return crc
}

func GetCrc16(val int64) uint16 {
	return crc16CCITT([]byte(strconv.FormatInt(val, 10))) % 16384
}
