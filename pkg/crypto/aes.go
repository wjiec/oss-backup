package crypto

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/binary"
	"io"
)

type AesConfig struct {
	Password string `json:"password"`
}

type Aes struct {
	key []byte
	iv  []byte
}

func (a *Aes) Encrypt(src []byte) []byte {
	block, err := aes.NewCipher(a.key)
	if err != nil {
		panic(err)
	}

	cAes := cipher.NewCBCEncrypter(block, a.iv)

	buf := src
	if pad := len(buf) % aes.BlockSize; pad != 0 {
		buf = append(buf, bytes.Repeat([]byte{byte(aes.BlockSize - pad)}, aes.BlockSize-pad)...)
	}

	dst := make([]byte, len(buf))
	cAes.CryptBlocks(dst, buf)

	return append(uin32ToBytes(uint32(len(dst))), dst...)
}

func (a *Aes) Decrypt(src []byte) (int, []byte) {
	if len(src) < 4 {
		return 0, nil
	}

	block, err := aes.NewCipher(a.key)
	if err != nil {
		panic(err)
	}

	cnt := bytesToUint32(src[:4])
	if len(src) < int(cnt+4) {
		return 0, nil
	}

	dst := make([]byte, cnt)
	cAes := cipher.NewCBCDecrypter(block, a.iv)
	cAes.CryptBlocks(dst, src[4:cnt+4])

	if len(dst) != 0 && dst[len(dst)-1] < aes.BlockSize {
		if len(dst) == 16384 {
			return int(cnt + 4), dst
		}
		dst = dst[:len(dst)-int(dst[len(dst)-1])]
	}
	return int(cnt + 4), dst
}

func (a *Aes) EncryptToBase64(src []byte) string {
	return base64.URLEncoding.EncodeToString(a.Encrypt(src))
}

func (a *Aes) DecryptFromBase64(v string) []byte {
	if bs, err := base64.URLEncoding.DecodeString(v); err == nil {
		_, bs := a.Decrypt(bs)
		return bs
	}

	return nil
}

type proxyReader struct {
	aes    *Aes
	source io.Reader
}

func (r *proxyReader) Read(p []byte) (int, error) {
	buf := make([]byte, 16384)
	rn, err := r.source.Read(buf)
	if err != nil {
		return rn, err
	}

	total := 0
	bs := r.aes.Encrypt(buf[:rn])
	for bs != nil {
		if n, err := bytes.NewReader(bs).Read(p); err != nil || n == len(bs) {
			return n, err
		} else if n != len(bs) {
			total += n
			bs = bs[n:]
		}
	}
	return total, nil
}

func (a *Aes) ProxyReader(src io.Reader) io.Reader {
	return &proxyReader{source: src, aes: a}
}

type proxyWriter struct {
	aes    *Aes
	eof    bool
	source *bufio.Writer

	buf []byte
}

func (w *proxyWriter) Write(p []byte) (n int, err error) {
	w.buf = append(w.buf, p...)
	defer func() { _ = w.source.Flush() }()

	for {
		dn, bs := w.aes.Decrypt(w.buf)
		if bs == nil {
			return len(p), nil
		}

		w.buf = w.buf[dn:]
		if wn, err := w.source.Write(bs); err != nil {
			return wn, err
		}
	}
}

func (a *Aes) ProxyWriter(src io.Writer) io.Writer {
	return &proxyWriter{aes: a, source: bufio.NewWriter(src)}
}

func NewAes(cfg *AesConfig) (*Aes, error) {
	key, iv := bytesToKey([]byte(cfg.Password))
	return &Aes{key: key, iv: iv}, nil
}

const (
	aesKeyLength   = 32
	aesBlockLength = aes.BlockSize
)

func bytesToKey(data []byte) (key, iv []byte) {
	h := md5.New()

	var result, last []byte
	for len(result) < aesKeyLength+aesBlockLength {
		h.Write(append(last, data...))
		last = h.Sum(nil)
		result = append(result, last...)
	}
	return result[:aesKeyLength], result[aesKeyLength : aesKeyLength+aesBlockLength]
}

func uin32ToBytes(n uint32) []byte {
	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.BigEndian, n)

	return buf.Bytes()
}

func bytesToUint32(bs []byte) (n uint32) {
	_ = binary.Read(bytes.NewReader(bs), binary.BigEndian, &n)

	return
}
