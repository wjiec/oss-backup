package cmd

import (
	"fmt"
	"log"
	"oss-backup/pkg/conf"
	"oss-backup/pkg/crypto"
	"oss-backup/pkg/storage"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

func ListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "list all objects",
	}

	cmd.PersistentFlags().StringP("password", "", "", "password to encrypt filename")
	cmd.Run = doListCommand

	return cmd
}

func doListCommand(cmd *cobra.Command, args []string) {
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

	for item := range ch {
		fmt.Printf("%s -> %s(%s: %s)\n", item.ObjectKey, aes.DecryptFromBase64(item.Metadata.Filename()),
			time.Unix(item.Metadata.ModTime(), 0).Format(time.RFC3339), bytesCount(item.Metadata.FileSize()))
	}
}

func bytesCount(n int) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%dB", n)
	case n >= 1024 && n < 1024*1024:
		return fmt.Sprintf("%.2fKB", float64(n)/1024)
	case n >= 1024*1024 && n < 1024*1024*1024:
		return fmt.Sprintf("%.2fKB", float64(n)/1024/1024)
	case n >= 1024*1024*1024 && n < 1024*1024*1024*1024:
		return fmt.Sprintf("%.2fMB", float64(n)/1024/1024/1024)
	default:
		return fmt.Sprintf("%.2fGB", float64(n)/1024/1024/1024/1024)
	}
}
