package protocol

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strconv"
)

/*
 * ============================================================================
 * RESP (REdis Serialization Protocol) 协议实现
 * ============================================================================
 *
 * RESP 是 Redis 使用的协议，支持以下数据类型：
 * - 简单字符串 (Simple String): +OK\r\n
 * - 错误 (Error): -ERR message\r\n
 * - 整数 (Integer): :1000\r\n
 * - 批量字符串 (Bulk String): $5\r\nhello\r\n
 * - 数组 (Array): *2\r\n$5\r\nhello\r\n$5\r\nworld\r\n
 *
 * 【协议格式】
 * - 简单字符串: +<string>\r\n
 * - 错误: -<string>\r\n
 * - 整数: :<number>\r\n
 * - 批量字符串: $<length>\r\n<data>\r\n
 * - 数组: *<count>\r\n<elements>...
 */

var (
	ErrInvalidFormat = errors.New("invalid RESP format")
)

// RESPType RESP 数据类型
type RESPType byte

const (
	RESP_SIMPLE_STRING RESPType = '+'
	RESP_ERROR         RESPType = '-'
	RESP_INTEGER       RESPType = ':'
	RESP_BULK_STRING   RESPType = '$'
	RESP_ARRAY         RESPType = '*'
)

// RESPValue RESP 值
type RESPValue struct {
	Type  RESPType
	Str   string
	Int   int64
	Array []*RESPValue
	Null  bool // 用于 nil 批量字符串
}

// NewSimpleString 创建简单字符串
func NewSimpleString(s string) *RESPValue {
	return &RESPValue{
		Type: RESP_SIMPLE_STRING,
		Str:  s,
	}
}

// NewError 创建错误
func NewError(s string) *RESPValue {
	return &RESPValue{
		Type: RESP_ERROR,
		Str:  s,
	}
}

// NewInteger 创建整数
func NewInteger(i int64) *RESPValue {
	return &RESPValue{
		Type: RESP_INTEGER,
		Int:  i,
	}
}

// NewBulkString 创建批量字符串
func NewBulkString(s string) *RESPValue {
	return &RESPValue{
		Type: RESP_BULK_STRING,
		Str:  s,
		Null: false,
	}
}

// NewNullBulkString 创建空批量字符串
func NewNullBulkString() *RESPValue {
	return &RESPValue{
		Type: RESP_BULK_STRING,
		Null: true,
	}
}

// NewArray 创建数组
func NewArray(elements []*RESPValue) *RESPValue {
	return &RESPValue{
		Type:  RESP_ARRAY,
		Array: elements,
	}
}

// Encode 编码为 RESP 格式
func (v *RESPValue) Encode() []byte {
	var buf bytes.Buffer

	switch v.Type {
	case RESP_SIMPLE_STRING:
		buf.WriteByte('+')
		buf.WriteString(v.Str)
		buf.WriteString("\r\n")

	case RESP_ERROR:
		buf.WriteByte('-')
		buf.WriteString(v.Str)
		buf.WriteString("\r\n")

	case RESP_INTEGER:
		buf.WriteByte(':')
		buf.WriteString(strconv.FormatInt(v.Int, 10))
		buf.WriteString("\r\n")

	case RESP_BULK_STRING:
		if v.Null {
			buf.WriteString("$-1\r\n")
		} else {
			buf.WriteByte('$')
			buf.WriteString(strconv.Itoa(len(v.Str)))
			buf.WriteString("\r\n")
			buf.WriteString(v.Str)
			buf.WriteString("\r\n")
		}

	case RESP_ARRAY:
		buf.WriteByte('*')
		buf.WriteString(strconv.Itoa(len(v.Array)))
		buf.WriteString("\r\n")
		for _, elem := range v.Array {
			buf.Write(elem.Encode())
		}
	}

	return buf.Bytes()
}

// Decode 从 Reader 解码 RESP 值
func Decode(reader *bufio.Reader) (*RESPValue, error) {
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	if len(line) < 2 || line[len(line)-2] != '\r' {
		return nil, ErrInvalidFormat
	}

	line = line[:len(line)-2] // 移除 \r\n

	if len(line) == 0 {
		return nil, ErrInvalidFormat
	}

	switch line[0] {
	case '+':
		// 简单字符串
		return &RESPValue{
			Type: RESP_SIMPLE_STRING,
			Str:  string(line[1:]),
		}, nil

	case '-':
		// 错误
		return &RESPValue{
			Type: RESP_ERROR,
			Str:  string(line[1:]),
		}, nil

	case ':':
		// 整数
		i, err := strconv.ParseInt(string(line[1:]), 10, 64)
		if err != nil {
			return nil, ErrInvalidFormat
		}
		return &RESPValue{
			Type: RESP_INTEGER,
			Int:  i,
		}, nil

	case '$':
		// 批量字符串
		length, err := strconv.Atoi(string(line[1:]))
		if err != nil {
			return nil, ErrInvalidFormat
		}

		if length == -1 {
			// NULL 批量字符串
			return &RESPValue{
				Type: RESP_BULK_STRING,
				Null: true,
			}, nil
		}

		data := make([]byte, length)
		if _, err := io.ReadFull(reader, data); err != nil {
			return nil, err
		}

		// 读取 \r\n
		crlf := make([]byte, 2)
		if _, err := io.ReadFull(reader, crlf); err != nil {
			return nil, err
		}
		if crlf[0] != '\r' || crlf[1] != '\n' {
			return nil, ErrInvalidFormat
		}

		return &RESPValue{
			Type: RESP_BULK_STRING,
			Str:  string(data),
			Null: false,
		}, nil

	case '*':
		// 数组
		count, err := strconv.Atoi(string(line[1:]))
		if err != nil {
			return nil, ErrInvalidFormat
		}

		if count == -1 {
			// NULL 数组
			return &RESPValue{
				Type:  RESP_ARRAY,
				Array: nil,
			}, nil
		}

		array := make([]*RESPValue, count)
		for i := 0; i < count; i++ {
			elem, err := Decode(reader)
			if err != nil {
				return nil, err
			}
			array[i] = elem
		}

		return &RESPValue{
			Type:  RESP_ARRAY,
			Array: array,
		}, nil

	default:
		return nil, ErrInvalidFormat
	}
}

// DecodeFromBytes 从字节数组解码
func DecodeFromBytes(data []byte) (*RESPValue, error) {
	reader := bufio.NewReader(bytes.NewReader(data))
	return Decode(reader)
}

// ToString 转换为字符串（用于命令参数）
func (v *RESPValue) ToString() string {
	if v.Type == RESP_BULK_STRING {
		return v.Str
	}
	if v.Type == RESP_SIMPLE_STRING {
		return v.Str
	}
	return ""
}

// ToInt 转换为整数
func (v *RESPValue) ToInt() int64 {
	if v.Type == RESP_INTEGER {
		return v.Int
	}
	return 0
}

// IsArray 是否是数组
func (v *RESPValue) IsArray() bool {
	return v.Type == RESP_ARRAY
}

// GetArray 获取数组元素
func (v *RESPValue) GetArray() []*RESPValue {
	if v.Type == RESP_ARRAY {
		return v.Array
	}
	return nil
}
