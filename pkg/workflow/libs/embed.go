package libs

import _ "embed"

//go:embed string_lib.gox
var StringLibCode string

//go:embed math_lib.gox
var MathLibCode string

//go:embed time_lib.gox
var TimeLibCode string

//go:embed crypto_lib.gox
var CryptoLibCode string

//go:embed regex_lib.gox
var RegexLibCode string

//go:embed validation_lib.gox
var ValidationLibCode string

//go:embed array_lib.gox
var ArrayLibCode string

//go:embed http_lib.gox
var HttpLibCode string

// GetAllLibsCode 获取所有库的代码
func GetAllLibsCode() string {
	return StringLibCode + "\n" +
		MathLibCode + "\n" +
		TimeLibCode + "\n" +
		CryptoLibCode + "\n" +
		RegexLibCode + "\n" +
		ValidationLibCode + "\n" +
		ArrayLibCode + "\n" +
		HttpLibCode
}
