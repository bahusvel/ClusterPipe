package ccssh

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	CONNECTION_TIMEOUT = 30 * time.Second
)

var (
	IDENTITY        = os.Getenv("HOME") + "/.ssh/id_rsa"
	connectionCache = map[string]*ssh.Client{}
	mapMutex        = sync.RWMutex{}
)

type CommandInterface interface {
	CombinedOutput() ([]byte, error)
	Output() ([]byte, error)
	Run() error
	Start() error
	StderrPipe() (io.ReadCloser, error)
	StdinPipe() (io.WriteCloser, error)
	StdoutPipe() (io.ReadCloser, error)
	Wait() error
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

type SessionCommand struct {
	*ssh.Session
	commandString string
}

func NewSessionCommand(client *ssh.Client, commandString string) (*SessionCommand, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	return &SessionCommand{Session: session, commandString: commandString}, nil
}

func (c *SessionCommand) CombinedOutput() ([]byte, error) {
	defer c.Session.Close()
	return c.Session.CombinedOutput(c.commandString)
}
func (c *SessionCommand) Output() ([]byte, error) {
	defer c.Session.Close()
	var errBuf bytes.Buffer
	c.Session.Stderr = &errBuf
	out, err := c.Session.Output(c.commandString)
	if err != nil {
		return []byte{}, errors.New(err.Error() + "(" + errBuf.String() + ")")
	}
	return out, nil
}
func (c *SessionCommand) Run() error {
	defer c.Session.Close()
	var errBuf bytes.Buffer
	c.Session.Stderr = &errBuf
	err := c.Session.Run(c.commandString)
	if err != nil {
		return errors.New(err.Error() + "(" + errBuf.String() + ")")
	}
	return nil
}
func (c *SessionCommand) Start() error {
	// NOTE start is async so how can I know when to close the session? Call wait in a goroutine!
	err := c.Session.Start(c.commandString)
	if err != nil {
		c.Session.Close()
		return err
	}
	go c.Session.Wait()
	return nil
}
func (c *SessionCommand) StderrPipe() (io.ReadCloser, error) {
	reader, err := c.Session.StderrPipe()
	return nopCloser{reader}, err
}
func (c *SessionCommand) StdoutPipe() (io.ReadCloser, error) {
	reader, err := c.Session.StdoutPipe()
	return nopCloser{reader}, err
}
func (c *SessionCommand) Wait() error {
	defer c.Session.Close()
	return c.Session.Wait()
}

type SystemCommand struct {
	*exec.Cmd
}

func (c *SystemCommand) Output() ([]byte, error) {
	var errBuf bytes.Buffer
	c.Cmd.Stderr = &errBuf
	out, err := c.Cmd.Output()
	if err != nil {
		return []byte{}, errors.New(err.Error() + "(" + errBuf.String() + ")")
	}
	return out, nil
}

func (c *SystemCommand) Run() error {
	var errBuf bytes.Buffer
	c.Cmd.Stderr = &errBuf
	err := c.Cmd.Run()
	if err != nil {
		return errors.New(err.Error() + "(" + errBuf.String() + ")")
	}
	return nil
}

func PasswordCommand(host string, password string, command string) (string, error) {
	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
	}
	client, err := ssh.Dial("tcp", host+":22", config)
	if err != nil {
		return "", err
	}
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	out, err := session.Output(command)
	if err != nil {
		return string(out), err
	}
	return string(out), err
}

func sshClient(host string) (*ssh.Client, error) {
	mapMutex.RLock()
	client, ok := connectionCache[host]
	if ok != false {
		mapMutex.RUnlock()
		return client, nil
	}
	mapMutex.RUnlock()
	key, err := ioutil.ReadFile(IDENTITY)
	if err != nil {
		log.Fatal(err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatal(err)
	}
	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		Timeout: CONNECTION_TIMEOUT,
	}
	client, err = ssh.Dial("tcp", host+":22", config)
	if err == nil {
		mapMutex.Lock()
		connectionCache[host] = client
		mapMutex.Unlock()
	}
	return client, err
}

func CopyID(from string, idPath string, to string, toPassword string) error {
	cmd := Command(from, "cat", idPath)
	data, err := cmd.Output()
	if err != nil {
		return err
	}
	_, err = PasswordCommand(to, toPassword, fmt.Sprintf("echo '%s' >> ~/.ssh/authorized_keys", string(data)))
	if err != nil {
		return err
	}
	return nil
}

func GenerateID(host string, path string) error {
	if RemoteExists(host, path, "-f") {
		return nil
	}
	cmd := Command(host, "ssh-keygen", "-q", "-N", "\"\"", "-f", path)
	fmt.Println(path)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func SendFile(server string, localfile string, remotefile string) error {
	cmd := exec.Command("scp", "-r", "-i", IDENTITY, localfile, fmt.Sprintf("root@%s:%s", server, remotefile))
	return cmd.Run()
}

func Command(host string, command string, args ...string) CommandInterface {
	commandString := command
	for _, arg := range args {
		commandString += " " + arg
	}
	if host == "localhost" {
		return &SystemCommand{exec.Command("sh", "-c", commandString)}
	} else {
		client, err := sshClient(host)
		if err != nil {
			log.Println("Unable to connect to host", host, err)
			return nil
		}
		command, err := NewSessionCommand(client, commandString)
		if err != nil {
			log.Println("Unable to issue new session", host, err)
			return nil
		}
		return command
	}
}

func RemoteExists(host string, path string, flag string) bool {
	commandString := fmt.Sprintf("[ %s %s ] && echo '1' || echo '0'", flag, path)
	cmd := Command(host, commandString)
	data, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	if data[0] == byte('1') {
		return true
	}
	return false
}
