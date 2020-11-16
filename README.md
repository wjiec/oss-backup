# oss-backup
backup with encrypted using aes into AliyunOSS by golang


### Quick start

First, clone repo into local directory, and just run go build.
```shell
git clone https://github.com/wjiec/oss-backup
cd oss-backup
go build cmd/oss-backup/main.go
```

And, typing `./main --help` to getting help
```plain
Usage:
  oss-backup [command]

Available Commands:
  backup      upload files to remote
  config      Print and manage bucket
  download    download the object into local
  help        Help about any command
  ls          list all objects

Flags:
  -c, --config string   the configure to loading (default "~/.oss_backup.json")
  -h, --help            help for oss-backup
  -u, --use string      use the bucket as default

Use "oss-backup [command] --help" for more information about a command.
```


### License

oss-backup is licensed under the [MIT license](https://github.com/wjiec/oss-backup/blob/master/LICENSE).