package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/robfig/cron"
)

type connectInfo struct {
	addr    *string
	port    *int
	enabled *bool
}

var (
	h               *bool
	schedule        *string
	localParentDir  *string
	retention       *int
	remoteParentDir *string
	host            *string
	user            *string
	password        *string
	s               = []string{"mongo", "es"}
	dbArgs          = make(map[string]*connectInfo)
	d               = make(map[string]db)
)

var (
	Info  *log.Logger
	Error *log.Logger
)

func init() {
	for _, v := range s {
		dbArgs[v] = &connectInfo{}
	}
}

func init() {
	h = flag.Bool("h", false, "print help")
	schedule = flag.String("schedule", "1 1 3 * * *", "cron schedule")
	localParentDir = flag.String("local_parent_dir", "/data/backup", "parent backup directory")
	retention = flag.Int("retention", 3, "retention days")
	remoteParentDir = flag.String("remote_parent_dir", "/home/stt", "remote parent directory")
	host = flag.String("host", "localhost:22", "sftp server IP:port")
	user = flag.String("user", "inu", "sftp username")
	password = flag.String("password", "password", "sftp password")
	dbArgs["mongo"].addr = flag.String("mongo_addr", "mongodb", "mongo address")
	dbArgs["mongo"].port = flag.Int("mongo_port", 27017, "mongo port")
	dbArgs["mongo"].enabled = flag.Bool("mongo_enable", true, "enable mongo backup or not")
	dbArgs["es"].addr = flag.String("es_addr", "es-client", "es address")
	dbArgs["es"].port = flag.Int("es_port", 9200, "es port")
	dbArgs["es"].enabled = flag.Bool("es_enable", true, "enable es backup or not")
}

func init() {
	f := log.Ldate | log.Ltime | log.Lshortfile
	Info = log.New(os.Stdout, "[INFO] ", f)
	Error = log.New(os.Stderr, "[ERROR] ", f)
}

func main() {
	flag.Parse()

	if *h {
		flag.Usage()
	}

	Info.Println("sftp mode:")
	var c STTFileMode
	c = NewSftp(*host, *user, *password, *remoteParentDir)
	defer c.Close()

	var out, archive string
	var err error
	remoteSubDir := "/backup"

	myCron := cron.New()
	spec := *schedule
	myCron.AddFunc(spec, func() {
		for _, v := range s {
			if *dbArgs[v].enabled {
				Info.Printf("Start %v backup:\n", v)
				backupLocalDir := fmt.Sprintf("%v/%v", *localParentDir, v)

				err = os.MkdirAll(backupLocalDir, os.ModePerm)
				if err != nil {
					Error.Println(err)
				}

				if err = c.prepareAllDir(remoteSubDir); err != nil {
					Error.Println(err)
				}

				switch v {
				case "mongo":
					d[v] = NewMongo(v, *dbArgs[v].addr, backupLocalDir, *dbArgs[v].port, *retention)
				case "es":
					d[v] = NewEs(v, *dbArgs[v].addr, backupLocalDir, *dbArgs[v].port, *retention)
				default:
					fmt.Println("Nothing to do")
					return
				}

				out, err = d[v].check()
				if err != nil {
					Error.Println(err)
				}
				if out != "" {
					Info.Println(out)
				}

				archive, out, err = d[v].dump()
				if err != nil {
					Error.Println(err)
				}
				if out != "" {
					Info.Println(out)
				}

				err = d[v].cleanup()
				if err != nil {
					Error.Println(err)
				}

				err = c.upload(archive, remoteSubDir, v)
				if err != nil {
					Error.Println(err)
				}
			}
		}
	})
	myCron.Start()
	select {}
}
