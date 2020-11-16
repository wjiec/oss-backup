package storage

import (
	"context"
	"io"
	"log"
	"math/rand"
	"oss-backup/pkg/conf"
	"strings"
	"sync"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type AliYunConfig struct {
	Endpoint        string `json:"endpoint"`
	AccessKeyId     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	Bucket          string `json:"bucket"`
	ObjectPrefix    string `json:"object_prefix"`
}

func (cfg *AliYunConfig) Validate() bool {
	switch "" {
	case cfg.Endpoint, cfg.AccessKeyId, cfg.AccessKeySecret, cfg.Bucket:
		return false
	}
	return true
}

type AliYunOSS struct {
	cfg    *conf.Bucket
	bucket *oss.Bucket
}

type progress struct {
	item *Item
}

func (p *progress) ProgressChanged(event *oss.ProgressEvent) {
	switch event.EventType {
	case oss.TransferStartedEvent:
		log.Printf("%s started", p.item.Filename)
	case oss.TransferDataEvent:
		percent := float64(event.ConsumedBytes) / float64(p.item.FileSize) * 100
		if percent > 100 {
			percent = 100
		}

		if percent != 100 {
			if rand.Intn(100) != 1 {
				break
			}
		}

		log.Printf("%s ... %.2f%%", p.item.Filename, percent)
	case oss.TransferCompletedEvent:
		log.Printf("%s completed", p.item.Filename)
	case oss.TransferFailedEvent:
		log.Printf("%s failed", p.item.Filename)
	}
}

func (ao *AliYunOSS) Exists(key string) bool {
	ok, err := ao.bucket.IsObjectExist(ao.genObjectKey(key))
	return ok && err == nil
}

func (ao *AliYunOSS) Metadata(key string) Metadata {
	md := make(Metadata)
	if props, err := ao.bucket.GetObjectDetailedMeta(ao.genObjectKey(key)); err == nil {
		for k, vs := range props {
			if len(vs) != 0 {
				md[k] = vs[0]
			}
		}
	}

	return md
}

func (ao *AliYunOSS) ListObject(prefix string) chan *Item {
	marker := ""
	ch := make(chan *Item, 1024)
	wg := sync.WaitGroup{}

	go func() {
		for {
			opts := []oss.Option{oss.Marker(marker), oss.MaxKeys(1000)}
			if prefix != "" {
				opts = append(opts, oss.Prefix(prefix))
			}

			res, err := ao.bucket.ListObjects(opts...)
			if err != nil {
				log.Fatal(err)
			}

			for _, obj := range res.Objects {
				wg.Add(1)
				item := &Item{ObjectKey: obj.Key, FileSize: obj.Size}
				go func(item *Item) {
					item.Metadata = ao.Metadata(item.ObjectKey)
					item.FileSize = int64(item.Metadata.FileSize())
					ch <- item
					wg.Done()
				}(item)
			}

			if res.IsTruncated {
				marker = res.NextMarker
			} else {
				break
			}
		}

		wg.Wait()
		close(ch)
	}()

	return ch
}

func (ao *AliYunOSS) Upload(ctx context.Context, item *Item, data io.Reader, metadata Metadata) error {
	opts := []oss.Option{oss.Progress(&progress{item: item})}
	for k, v := range metadata {
		opts = append(opts, oss.Meta(k, v))
	}

	return ao.bucket.PutObject(ao.genObjectKey(item.ObjectKey), data, opts...)
}

func (ao *AliYunOSS) Download(ctx context.Context, item *Item, w io.Writer) error {
	rd, err := ao.bucket.GetObject(item.ObjectKey, oss.AcceptEncoding("gzip"), oss.Progress(&progress{item: item}))
	if err != nil {
		return err
	}

	_, err = io.Copy(w, rd)
	return err
}

var (
	trim = func(s string) string { return strings.Trim(s, "/\\") }
)

func (ao *AliYunOSS) genObjectKey(key string) string {
	return trim(trim(ao.cfg.ObjectPrefix) + "/" + trim(key))
}

func NewAliYunOSS(cfg *conf.Bucket) (*AliYunOSS, error) {
	client, err := oss.New(cfg.Endpoint, cfg.AccessKeyId, cfg.AccessKeySecret)
	if err != nil {
		return nil, err
	}

	bucket, err := client.Bucket(cfg.BucketName)
	if err != nil {
		return nil, err
	}

	return &AliYunOSS{cfg: cfg, bucket: bucket}, nil
}
