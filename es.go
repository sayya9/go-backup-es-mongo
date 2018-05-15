package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	sh "github.com/codeskyblue/go-sh"
	"github.com/mholt/archiver"
	"github.com/pkg/errors"
)

type es struct {
	name           string
	addr           string
	port           int
	backupLocalDir string
	retention      int
}

func NewEs(name, addr, backupLocalDir string, port, retention int) *es {
	e := &es{
		name:           name,
		addr:           addr,
		port:           port,
		backupLocalDir: backupLocalDir,
		retention:      retention,
	}
	return e
}

func (e es) check() (string, error) {
	fullAddr := fmt.Sprintf("http://%v:%v", e.addr, strconv.Itoa(e.port))
	resp, err := http.Get(fullAddr)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (e *es) dump() (string, string, error) {
	t := time.Now()
	date := fmt.Sprintf("%v-%v-%v", t.Year(), int(t.Month()), t.Day())

	archive := fmt.Sprintf("%v/%v-%v.gz", e.backupLocalDir, e.name, date)
	fullAddr := fmt.Sprintf("http://%v:%v", e.addr, strconv.Itoa(e.port))

	// Register a snapshot repository with the name my_backup
	data := `{"type":"fs","settings":{"location":"/snapshot/backups/my_backup","compress":true}}`
	jsonStr := []byte(data)
	err := request(fullAddr+"/_snapshot/my_backup", bytes.NewBuffer(jsonStr), "PUT")
	if err != nil {
		return "", "", err
	}

	// Create a snapshot with the name snapshot_1 in the repository my_backup
	data = ""
	jsonStr = []byte(data)
	err = request(fullAddr+"/_snapshot/my_backup/snapshot_1?wait_for_completion=true", bytes.NewBuffer(jsonStr), "PUT")
	if err != nil {
		return "", "", err
	}

	// Compressed with gzip format from the es snapshot
	err = archiver.TarGz.Make(archive, []string{"/snapshot/backups/my_backup"})
	if err != nil {
		return "", "", err
	}

	// Delete current snapshot
	err = request(fullAddr+"/_snapshot/my_backup/snapshot_1", nil, "DELETE")
	if err != nil {
		return "", "", err
	}

	return archive, "", err
}

func (e es) cleanup() error {
	gz := fmt.Sprintf("cd %v && rm -f $(ls -1t %v*.gz | tail -n +%v)", e.backupLocalDir, e.name, e.retention+1)
	err := sh.Command("sh", "-c", gz).Run()
	if err != nil {
		return errors.Wrapf(err, "removing old gz files from %v failed", e.backupLocalDir)
	}

	return nil
}

func request(url string, data io.Reader, method string) error {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, data)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, _ := ioutil.ReadAll(resp.Body)
	Info.Println(string(out))

	return nil
}
