package main

import (
	"fmt"
	"net"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type sftpMode struct {
	remoteParentDir string
	Host            string
	User            string
	Password        string
	SftpClient      connectInterface
}

func NewSftp(host, user, password, remoteParentDir string) *sftpMode {
	Info.Println("Remote Directory: ", remoteParentDir)
	c := &sftpMode{
		remoteParentDir: remoteParentDir,
		Host:            host,
		User:            user,
		Password:        password,
		SftpClient:      getSftpConn(host, user, password),
	}
	return c
}

func (c *sftpMode) prepareAllDir(subPath string) error {
	err := c.createRemoteDir(subPath)
	if err != nil {
		return errors.Wrapf(err, "can't create %v directory", c.remoteParentDir+subPath)
	}
	return nil
}

func (c *sftpMode) Close() {
	defer c.SftpClient.Close()
}

func (c *sftpMode) createRemoteDir(subPath string) error {
	p := c.remoteParentDir + subPath
	_, statErr := c.SftpClient.Stat(p)
	if statErr != nil {
		Info.Println("Create remote Directory: ", p)
		if err := c.SftpClient.Mkdir(p); err != nil {
			return err
		}
	}
	return nil
}

func (c *sftpMode) upload(localFilePath, subDir, dbName string) error {
	Info.Println("Start upload to sftp")
	srcFile, err := os.Open(localFilePath)
	if err != nil {
		Error.Println(err)
	}
	defer srcFile.Close()

	remoteFileName := fmt.Sprintf("%v.gz", dbName)
	dstFile, err := c.SftpClient.Create(path.Join(c.remoteParentDir, subDir, remoteFileName))
	if err != nil {
		Error.Println(err)
	}
	defer dstFile.Close()

	buf := make([]byte, 1024)
	for {
		n, _ := srcFile.Read(buf)
		if n == 0 {
			break
		}
		_, err = dstFile.Write(buf)
		if err != nil {
			Error.Println(err)
		}
	}

	Info.Println("Upload successfully")
	return nil
}

func getSSHConn(host, user, password string) (*ssh.Client, error) {
	var auths []ssh.AuthMethod
	if aconn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		auths = append(auths, ssh.PublicKeysCallback(agent.NewClient(aconn).Signers))
	}
	if password != "" {
		auths = append(auths, ssh.Password(password))
	}

	config := ssh.ClientConfig{
		User: user,
		Auth: auths,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	conn, err := ssh.Dial("tcp", host, &config)
	if err != nil {
		Error.Fatalf("unable to connect to [%s]: %v", host, err)
		return nil, err
	}
	Info.Println("connected to host: ", host)
	return conn, nil
}

func getSftpConn(host, user, password string) *sftp.Client {
	sshClient, err := getSSHConn(host, user, password)
	if err != nil {
		Error.Fatal(err)
	}
	sftpc, err := sftp.NewClient(sshClient)
	if err != nil {
		Error.Fatalf("unable to start sftp subsytem: %v", err)
	}
	return sftpc
}
