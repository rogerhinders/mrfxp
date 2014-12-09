package fxp

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

type FTPError struct {
	description string
}

type DirItem struct {
	permission string
	owner      string
	group      string
	size       int
	date       string
	name       string
}

type EncryptionType int

const (
	PLAIN EncryptionType = iota
	TLS
	SSL
)

func (e *FTPError) Error() string {
	return fmt.Sprintf("FTP ERROR: %s", e.description)
}

type FTPCredentials struct {
	username string
	password string
}

type FTPSite struct {
	hostname string
	port     int
}
type FTPClient struct {
	credentials FTPCredentials
	site        FTPSite
	conn        net.Conn
	encryption  EncryptionType
	portString  string
	logControl  bool
}

type FTPMessage struct {
	buffer bytes.Buffer
}

func (m *FTPMessage) Write(p []byte) (n int, err error) {
	return m.buffer.Write(p)
}

func (m *FTPMessage) String() string {
	return m.buffer.String()
}

func (m *FTPMessage) Bytes() []byte {
	return m.buffer.Bytes()
}

func (m *FTPMessage) ResponseCode() int {
	s := m.buffer.String()

	return (int(s[0])-0x30)*100 + (int(s[1])-0x30)*10 + int(s[2]) - 0x30
}

func (m *FTPMessage) GetLines() []string {
	data := m.String()
	var line bytes.Buffer
	var ret []string

	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			ret = append(ret, line.String())
			line.Reset()
		} else if data[i] == '\r' {
			continue
		} else {
			line.WriteString(string(data[i : i+1]))
		}
	}

	return ret
}

func (c *FTPSite) HostString() string {
	return fmt.Sprintf("%s:%d", c.hostname, c.port)
}

func (c *FTPClient) SetInfo(username string, password string, hostname string, port int, encryption EncryptionType) {
	c.credentials = FTPCredentials{username: username, password: password}
	c.site = FTPSite{hostname: hostname, port: port}
	c.encryption = encryption
	c.logControl = false
}

func (c *FTPClient) Connect() error {
	conn, err := net.Dial("tcp", c.site.HostString())
	c.conn = conn

	if err != nil {
		return err
	}

	msg, err := c.controlRecv()

	if err != nil {
		return err
	}

	if msg.ResponseCode() != 220 {
		return &FTPError{description: "initial response not valid."}
	}
	//tls.Dial()
	switch c.encryption {
	case TLS:
		c.controlSend("AUTH TLS\n")
		msg, err = c.controlRecv()

		if err != nil {
			return err
		}

		if msg.ResponseCode() != 234 {
			return &FTPError{description: "Auth TLS error."}
		}

		var config tls.Config
		config.InsecureSkipVerify = true

		tlsconn := tls.Client(c.conn, &config)
		err = tlsconn.Handshake()

		if err != nil {
			return err
		}

		c.conn = tlsconn
	case SSL:
		//TODO: ADD SSL SUPPORT
	case PLAIN:
	}

	c.controlSend(fmt.Sprintf("USER %s\n", c.credentials.username))
	msg, err = c.controlRecv()

	if err != nil {
		return err
	}

	if msg.ResponseCode() != 331 {
		return &FTPError{description: "invalid username."}
	}

	c.controlSend(fmt.Sprintf("PASS %s\n", c.credentials.password))
	msg, err = c.controlRecv()

	if err != nil {
		return err
	}

	if msg.ResponseCode() != 230 {
		return &FTPError{description: "invalid password."}
	}

	return nil
}

func (c *FTPClient) Close() error {
	c.controlSend("QUIT\n")
	c.controlRecv()
	return c.conn.Close()
}

func (c *FTPClient) controlSend(msg string) {
	if c.logControl {
		fmt.Printf(msg)
	}

	c.conn.Write([]byte(msg))
}

func (c *FTPClient) controlRecv() (*FTPMessage, error) {
	var msg FTPMessage
	var buf [bytes.MinRead]byte
	done := false
	currentLine := 0
	currentByte := 0
	var na, nb, nc int

	for done == false {
		n, err := c.conn.Read(buf[0:])

		if err != nil {
			return nil, err
		}

		msg.Write(buf[0:n])

		for i := 0; i < n; i++ {
			if buf[i] == '\n' {
				tmpBytes := msg.Bytes()

				//verify line starts with a code before doing anything else
				na = int(tmpBytes[currentLine] - 0x30)
				nb = int(tmpBytes[currentLine+1] - 0x30)
				nc = int(tmpBytes[currentLine+2] - 0x30)

				if (na < 10) && (nb < 10) && (nc < 10) && (tmpBytes[currentLine+3] != '-') {
					done = true
				}

				currentLine = currentByte + 1
			}
			currentByte++
		}
	}

	if c.logControl {
		fmt.Printf(msg.String())
	}

	return &msg, nil
}

func (c *FTPClient) EnterPasv(secure bool) error {
	if secure {
		c.controlSend("CPSV\n")
	} else {
		c.controlSend("PASV\n")
	}

	msg, err := c.controlRecv()

	if err != nil {
		return err
	}

	if msg.ResponseCode() != 227 {
		return &FTPError{description: "Failed entering passive mode."}
	}
	msgString := msg.String()

	start, end := 0, 0

	for i := 0; i < len(msgString); i++ {
		if msgString[i] == '(' {
			start = i + 1
		}

		if msgString[i] == ')' {
			end = i
		}
	}

	c.portString = msgString[start:end]
	return nil
}

func (c *FTPClient) Cwd(path string) error {
	c.controlSend(fmt.Sprintf("CWD %s\n", path))

	msg, err := c.controlRecv()

	if err != nil {
		return err
	}

	if msg.ResponseCode() != 250 {
		return &FTPError{description: "error changing directory."}
	}
	return nil
}

func (c *FTPClient) Dir() ([]DirItem, error) {
	c.controlSend("stat -l\n")
	msg, err := c.controlRecv()

	if err != nil {
		return nil, err
	}

	if msg.ResponseCode() != 213 {
		return nil, &FTPError{description: "dirlist error."}
	}
	//parse dir info
	var dir []DirItem

	lines := msg.GetLines()

	//if dir is empty
	if len(lines) < 3 {
		return dir, nil
	}

	for i := 2; i < len(lines); i++ {
		if (lines[i][0] - 0x30) < 10 {
			continue
		}
		fields := strings.Fields(lines[i])
		size, _ := strconv.Atoi(fields[4])

		item := DirItem{
			permission: fields[0],
			owner:      fields[2],
			group:      fields[3],
			size:       size,
			date:       fmt.Sprintf("%s %s %s", fields[5], fields[6], fields[7]),
			name:       fields[8],
		}
		dir = append(dir, item)
	}

	return dir, nil
}

func (c *FTPClient) FXPTo(dest *FTPClient, pathSrc string, pathDst string, file string, fileSize int64, secure bool) (int64, error) {

	e := c.EnterPasv(secure)

	if e != nil {
		return 0, e
	}

	dest.controlSend(fmt.Sprintf("PORT %s\n", c.portString))
	msg, err := dest.controlRecv()

	if err != nil {
		return 0, err
	}

	if msg.ResponseCode() != 200 {
		return 0, &FTPError{description: "error sending port cmd."}
	}

	err = c.Cwd(pathSrc)

	if err != nil {
		return 0, err
	}

	err = dest.Cwd(pathDst)

	if err != nil {
		return 0, err
	}

	dest.controlSend(fmt.Sprintf("STOR %s\n", file))
	msg, err = dest.controlRecv()

	if err != nil {
		return 0, err
	}

	if msg.ResponseCode() != 150 {
		return 0, &FTPError{description: "error initiating fxp."}
	}

	c.controlSend(fmt.Sprintf("RETR %s\n", file))
	startTime := time.Now().Unix()
	msg, err = c.controlRecv()

	if err != nil {
		return 0, err
	}

	if msg.ResponseCode() != 150 {
		return 0, &FTPError{description: "error initiating fxp."}
	}

	msg, err = c.controlRecv()

	if err != nil {
		return 0, err
	}

	if msg.ResponseCode() != 226 {
		return 0, &FTPError{description: "fxp failed."}
	}

	msg, err = dest.controlRecv()

	if err != nil {
		return 0, err
	}

	if msg.ResponseCode() != 226 {
		return 0, &FTPError{description: "fxp failed."}
	}

	duration := time.Now().Unix() - startTime

	return fileSize / duration, nil
}
