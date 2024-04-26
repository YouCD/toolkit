package file

import "encoding/base64"

// 四种 base64 编码实现方式  https://blog.51cto.com/lilongsy/6037599

func Base64EncodeStr(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}
func Base64EncodeByte(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
