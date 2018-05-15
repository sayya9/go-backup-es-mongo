package main

import (
	"os"

	"github.com/kr/fs"
	"github.com/pkg/sftp"
)

type db interface {
	check() (string, error)
	dump() (string, string, error)
	cleanup() error
}

type STTFileMode interface {
	prepareAllDir(string) error
	upload(string, string, string) error
	Close()
}

type connectInterface interface {
	Open(path string) (*sftp.File, error)
	Create(path string) (*sftp.File, error)
	Mkdir(path string) error
	Stat(p string) (os.FileInfo, error)
	Walk(root string) *fs.Walker
	Rename(oldname, newname string) error
	Remove(path string) error
	Close() error
}
