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
	"path"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

func DownloadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "download the object into local",
	}

	cmd.PersistentFlags().StringP("dir", "", "", "output dir")
	cmd.PersistentFlags().StringP("password", "", "", "password to encrypt filename")
	cmd.Run = doDownloadCommand

	return cmd
}

func doDownloadCommand(cmd *cobra.Command, args []string) {
	cfg := cmd.Context().Value("cfg").(*conf.Config)

	password, _ := cmd.Flags().GetString("password")
	if password == "" {
		log.Fatal("password is required")
	}

	aes, err := crypto.NewAes(&crypto.AesConfig{Password: password})
	if err != nil {
		log.Fatal(err)
	}

	oss, err := storage.NewAliYunOSS(cfg.GetBucket())
	if err != nil {
		log.Fatal(err)
	}
	uploader = oss

	if len(args) == 0 {
		args = append(args, "")
	}

	wg := sync.WaitGroup{}
	ch := make(chan *storage.Item, 1024)
	for _, name := range args {
		wg.Add(1)
		go func(name string) {
			for item := range oss.ListObject(name) {
				ch <- item
			}
			wg.Done()
		}(name)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	dir, _ := cmd.Flags().GetString("dir")
	cl := limiter.NewConcurrencyLimiter(1)
	for item := range ch {
		cl.Execute(func(args ...interface{}) {
			download(args[0].(string), args[1].(*storage.Item), args[2].(*crypto.Aes))
		}, dir, item, aes)
	}

	cl.Wait()
}

func download(dir string, item *storage.Item, aes *crypto.Aes) {
	filename := normalizeFilename(string(aes.DecryptFromBase64(item.Metadata.Filename())))
	if !utils.Exists(dir) {
		if err := os.MkdirAll(dir, os.ModeDir); err != nil {
			log.Printf("create dir: %s", err)
			return
		}
	}
	item.Filename = filename

	fp, err := os.Create(path.Join(dir, filename))
	if err != nil {
		log.Printf("open file: %s", err)
		return
	}

	buf := aes.ProxyWriter(fp)
	if err := uploader.Download(context.Background(), item, buf); err != nil {
		log.Printf("save file: %s", err)
		return
	}
}

func normalizeFilename(filename string) string {
	return strings.NewReplacer(":", "_", "\\", "_", "/", "_").Replace(filename)
}
