package cmd

import (
	"context"
	"log"
	"os"
	"oss-backup/pkg/conf"
	"oss-backup/pkg/crypto"
	"oss-backup/pkg/limiter"
	"oss-backup/pkg/storage"
	"oss-backup/pkg/utils"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"
)

var uploader storage.Uploader

func BackupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "upload files to remote",
	}

	cmd.PersistentFlags().StringP("prefix", "", "", "prefix of object key")
	cmd.PersistentFlags().StringP("password", "", "", "password to encrypt filename")
	cmd.PersistentFlags().IntP("max-concurrency", "", 5, "number of max upload concurrency")
	cmd.Run = doBackupCommand

	return cmd
}

func doBackupCommand(cmd *cobra.Command, args []string) {
	cfg := cmd.Context().Value("cfg").(*conf.Config)

	password, _ := cmd.Flags().GetString("password")
	if password == "" {
		log.Fatal("password is required")
	}

	aes, err := crypto.NewAes(&crypto.AesConfig{Password: password})
	if err != nil {
		log.Fatal(err)
	}

	bucket := cfg.GetBucket()
	prefix, _ := cmd.Flags().GetString("prefix")
	bucket.ObjectPrefix = prefix

	s, err := storage.NewAliYunOSS(bucket)
	if err != nil {
		log.Fatal(err)
	}
	uploader = s

	maxConcurrency, _ := cmd.Flags().GetInt("max-concurrency")
	cl := limiter.NewConcurrencyLimiter(maxConcurrency)

	wg := sync.WaitGroup{}
	for filename := range walk(args) {
		wg.Add(1)
		cl.Execute(func(args ...interface{}) {
			upload(args[0].(string), aes)
			wg.Done()
		}, filename)
	}

	wg.Wait()
}

func walk(paths []string) <-chan string {
	wg := sync.WaitGroup{}
	files := make(chan string)
	for _, path := range paths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			if ok, err := utils.IsDir(path); ok {
				_ = filepath.Walk(path, func(filename string, info os.FileInfo, err error) error {
					if err != nil || info.IsDir() {
						return nil
					}

					files <- filename
					return nil
				})
			} else if err == nil {
				files <- path
			}
		}(path)
	}

	go func() {
		wg.Wait()
		close(files)
	}()

	return files
}

func upload(filename string, aes *crypto.Aes) {
	stat, err := os.Stat(filename)
	if err != nil {
		log.Printf("unable to stat file %s", filename)
		return
	}

	key := utils.Md5(filename)
	if uploader.Exists(key) {
		log.Printf("object exists for file %s(%s)", filename, key)

		md := uploader.Metadata(key)
		if md.ModTime() == stat.ModTime().Unix() {
			log.Printf("file not modify %s(%s), SKIP", filename, key)
			return
		}
	}

	md := make(storage.Metadata)
	md.SetModTime(stat.ModTime().Unix())
	md.SetFilename(aes.EncryptToBase64([]byte(filename)))
	md.SetFileSize(int(stat.Size()))

	fp, err := os.Open(filename)
	if err != nil {
		log.Printf("cannot open file %s", err)
		return
	}

	item := &storage.Item{Filename: filename, ObjectKey: key, FileSize: stat.Size()}
	if err := uploader.Upload(context.Background(), item, aes.ProxyReader(fp), md); err != nil {
		log.Printf("upload file %s failed, cause by %s", filename, err)
	}
}
