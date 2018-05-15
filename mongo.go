package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/codeskyblue/go-sh"
	"github.com/pkg/errors"
)

type mongo struct {
	name           string
	addr           string
	port           int
	backupLocalDir string
	retention      int
}

func NewMongo(name, addr, backupLocalDir string, port, retention int) *mongo {
	m := &mongo{
		name:           name,
		addr:           addr,
		port:           port,
		backupLocalDir: backupLocalDir,
		retention:      retention,
	}
	return m
}

func (m mongo) check() (string, error) {
	output, err := sh.Command("sh", "-c", "mongodump --version").CombinedOutput()
	if err != nil {
		ex := ""
		if len(output) > 0 {
			ex = strings.Replace(string(output), "\n", " ", -1)
		}
		return "", errors.Wrapf(err, "mongodump failed %v", ex)
	}
	return strings.Replace(string(output), "\n", " ", -1), nil
}

func (m *mongo) dump() (string, string, error) {
	t := time.Now()
	date := fmt.Sprintf("%v-%v-%v", t.Year(), int(t.Month()), t.Day())

	archive := fmt.Sprintf("%v/%v-%v.gz", m.backupLocalDir, m.name, date)
	dump := fmt.Sprintf("mongodump --archive=%v --gzip --host %v --port %v", archive, m.addr, m.port)

	output, err := sh.Command("sh", "-c", dump).CombinedOutput()
	if err != nil {
		ex := ""
		if len(output) > 0 {
			ex = strings.Replace(string(output), "\n", " ", -1)
		}

		return "", "", errors.Wrapf(err, "mongodump log %v", ex)
	}

	return archive, strings.Replace(string(output), "\n", " ", -1), nil
}

func (m mongo) cleanup() error {
	gz := fmt.Sprintf("cd %v && rm -f $(ls -1t %v*.gz | tail -n +%v)", m.backupLocalDir, m.name, m.retention+1)
	err := sh.Command("sh", "-c", gz).Run()
	if err != nil {
		return errors.Wrapf(err, "removing old gz files from %v failed", m.backupLocalDir)
	}

	return nil
}
