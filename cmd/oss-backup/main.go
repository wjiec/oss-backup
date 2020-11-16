package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"oss-backup/internal/cmd"
	"oss-backup/pkg/conf"
	"oss-backup/pkg/utils"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

var cfg conf.Config

func main() {
	dfFilename, err := homedir.Expand("~/.oss_backup.json")
	if err != nil {
		log.Fatal(err)
	}

	root := &cobra.Command{Use: "oss-backup"}
	root.AddCommand(cmd.ConfigCommand())
	root.AddCommand(cmd.BackupCommand())
	root.AddCommand(cmd.ListCommand())
	root.AddCommand(cmd.DownloadCommand())

	root.PersistentFlags().StringP("config", "c", dfFilename, "the configure to loading")
	root.PersistentFlags().StringP("use", "u", "", "use the bucket as default")
	root.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
		filename, _ := cmd.Flags().GetString("config")
		if filename == "" {
			log.Fatal("please specify a configure file and continue")
		}

		cfg.Filename = filename
		if utils.Exists(filename) {
			bs, err := ioutil.ReadFile(filename)
			if err != nil {
				log.Fatal(err)
			}

			if len(bs) != 0 {
				if err := json.Unmarshal(bs, &cfg); err != nil {
					log.Fatal(err)
				}
			}

			if name, _ := cmd.Flags().GetString("use"); name != "" {
				if err := cfg.UseBucket(name); err != nil {
					log.Fatal(err)
				}
			}

			if cfg.GetBucket() == nil {
				log.Fatal("unknown default bucket, please re-run and specify by -u")
			}
		} else {
			if err := cfg.NewBucket(); err != nil {
				log.Fatal(err)
			}
		}
	}

	ctx := context.WithValue(context.Background(), "cfg", &cfg)
	if err := root.ExecuteContext(ctx); err != nil {
		log.Fatal(err)
	}
}
