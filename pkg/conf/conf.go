package conf

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

type Bucket struct {
	Alias           string `json:"alias"`
	Endpoint        string `json:"endpoint"`
	AccessKeyId     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	BucketName      string `json:"bucket_name"`
	RsaPrivateKey   string `json:"rsa_private"`

	ObjectPrefix string `json:"-"`
}

func (b *Bucket) Wizard() error {
	prompt("Endpoint: ", &b.Endpoint)
	prompt("AccessKeyId: ", &b.AccessKeyId)
	prompt("AccessKeySecret: ", &b.AccessKeySecret)
	prompt("BucketName: ", &b.BucketName)
	prompt("Alias: ", &b.Alias)

	if b.RsaPrivateKey == "" {
		k, err := rsa.GenerateKey(rand.Reader, rsaBitsSize)
		if err != nil {
			return err
		}

		b.RsaPrivateKey = base64.StdEncoding.EncodeToString(pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(k),
		}))
	}

	return nil
}

func (b *Bucket) DumpRsaPrivateKey(w io.Writer) error {
	bs, err := base64.StdEncoding.DecodeString(b.RsaPrivateKey)
	if err != nil {
		return err
	}

	return pem.Encode(w, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: bs,
	})
}

func (b *Bucket) DumpRsaPublicKey(w io.Writer) error {
	bs, err := base64.StdEncoding.DecodeString(b.RsaPrivateKey)
	if err != nil {
		return err
	}

	block, _ := pem.Decode(bs)
	pk, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil
	}

	return pem.Encode(w, &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&pk.PublicKey),
	})
}

func (b *Bucket) GetRsaPrivateKey() (string, error) {
	buf := bytes.Buffer{}
	if err := b.DumpRsaPrivateKey(&buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (b *Bucket) GetPublicKey() (string, error) {
	buf := bytes.Buffer{}
	if err := b.DumpRsaPublicKey(&buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (b *Bucket) NameIs(name string) bool {
	return b.BucketName == name || b.Alias == name
}

type Buckets []*Bucket

func (bs Buckets) Find(name string) *Bucket {
	for _, bucket := range bs {
		if bucket.NameIs(name) {
			return bucket
		}
	}

	return nil
}

type Config struct {
	Filename      string  `json:"-"`
	Buckets       Buckets `json:"buckets"`
	DefaultBucket string  `json:"default_bucket"`
}

const (
	rsaBitsSize = 2048
)

func (c *Config) NewBucket() error {
	var bucket Bucket
	if err := bucket.Wizard(); err != nil {
		return err
	}

	if c.Buckets.Find(bucket.Alias) != nil || c.Buckets.Find(bucket.BucketName) != nil {
		return errors.New("duplicated bucket name or alias")
	}

	c.DefaultBucket = bucket.Alias
	c.Buckets = append(c.Buckets, &bucket)
	return c.Save()
}

func (c *Config) FindBucket(name string) *Bucket {
	return c.Buckets.Find(name)
}

func (c *Config) RemoveBucket(name string) {
	var buckets Buckets
	for _, bucket := range c.Buckets {
		if !bucket.NameIs(name) {
			buckets = append(buckets, bucket)
		}
	}

	c.Buckets = buckets
	if c.GetBucket() == nil {
		c.DefaultBucket = ""
	}
}

func (c *Config) UseBucket(name string) error {
	if c.Buckets.Find(name) == nil {
		return errors.New("unknown bucket name")
	}

	c.DefaultBucket = name
	return c.Save()
}

func (c *Config) Save() error {
	bs, err := json.Marshal(c)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(c.Filename, bs, 0600)
}

func (c *Config) GetBucket() *Bucket {
	bucket := c.Buckets.Find(c.DefaultBucket)
	if bucket == nil && len(c.Buckets) != 0 {
		bucket = c.Buckets[0]
		_ = c.UseBucket(bucket.BucketName)
	}
	return bucket
}

func prompt(text string, v *string) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print(text)
		if scanner.Scan() {
			if s := scanner.Text(); strings.TrimSpace(s) != "" {
				*v = s
				break
			}
		}
	}
}
