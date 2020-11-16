package utils

import (
	"crypto/md5"
	"encoding/hex"
)

func Md5(val string) string {
	h := md5.New()
	h.Write([]byte(val))
	return hex.EncodeToString(h.Sum(nil))
}
