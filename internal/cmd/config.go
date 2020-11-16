package cmd

import (
	"fmt"
	"log"
	"os"
	"oss-backup/pkg/conf"

	"github.com/spf13/cobra"
)

func ConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Print and manage bucket",
	}

	cmd.PersistentFlags().BoolP("new", "", false, "create new bucket")
	cmd.PersistentFlags().BoolP("all", "a", true, "list all buckets")
	cmd.PersistentFlags().BoolP("dump-pem", "", false, "dump bucket rsa key")
	cmd.PersistentFlags().BoolP("delete", "d", false, "delete bucket")
	cmd.Run = doConfigCommand

	return cmd
}

func doConfigCommand(cmd *cobra.Command, args []string) {
	cfg := cmd.Context().Value("cfg").(*conf.Config)

	if create, _ := cmd.Flags().GetBool("new"); create {
		if err := cfg.NewBucket(); err != nil {
			log.Fatal(err)
		}

		os.Exit(0)
	}

	if dump, _ := cmd.Flags().GetBool("dump-pem"); dump {
		buckets := []string{cfg.DefaultBucket}
		if len(args) != 0 {
			buckets = args
		}

		for _, name := range buckets {
			if bucket := cfg.FindBucket(name); bucket != nil {
				fp, err := os.Create(bucket.BucketName + ".pem")
				if err != nil {
					log.Fatal(err)
				}

				if err := bucket.DumpRsaPrivateKey(fp); err != nil {
					log.Fatal(err)
				}
			}
		}

		os.Exit(0)
	}

	if remove, _ := cmd.Flags().GetBool("delete"); remove {
		for _, name := range args {
			cfg.RemoveBucket(name)
			fmt.Printf("Remove %s\n", name)
		}

		if err := cfg.Save(); err != nil {
			log.Fatal(err)
		}

		os.Exit(0)
	}

	buckets := conf.Buckets{cfg.GetBucket()}
	if list, _ := cmd.Flags().GetBool("all"); list {
		buckets = cfg.Buckets
	}

	for i, bucket := range buckets {
		if bucket == nil {
			continue
		}

		tag := ""
		if bucket.NameIs(cfg.DefaultBucket) {
			tag = "(Default)"
		}

		fmt.Printf("Alias: %s%s\n", bucket.Alias, tag)
		fmt.Printf("BucketName: %s\n", bucket.BucketName)
		fmt.Printf("Endpoint: %s\n", bucket.Endpoint)
		fmt.Printf("AccessKeyId: %s\n", bucket.AccessKeyId)
		fmt.Printf("AccessKeySecret: %s\n", bucket.AccessKeySecret)

		if len(buckets) != 1 && i != len(buckets)-1 {
			fmt.Println()
		}
	}
}
