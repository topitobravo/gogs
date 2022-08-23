// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Unknwon/com"

	"github.com/topitobravo/gogs/modules/log"
	"github.com/topitobravo/gogs/modules/process"
	"github.com/topitobravo/gogs/modules/setting"
)

const (
	// "### autogenerated by gitgos, DO NOT EDIT\n"
	_TPL_PUBLICK_KEY = `command="%s serv key-%d",no-port-forwarding,no-X11-forwarding,no-agent-forwarding,no-pty %s` + "\n"
)

var (
	ErrKeyAlreadyExist = errors.New("Public key already exist")
	ErrKeyNotExist     = errors.New("Public key does not exist")
)

var sshOpLocker = sync.Mutex{}

var (
	SshPath string // SSH directory.
	appPath string // Execution(binary) path.
)

// exePath returns the executable path.
func exePath() (string, error) {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		return "", err
	}
	return filepath.Abs(file)
}

// homeDir returns the home directory of current user.
func homeDir() string {
	home, err := com.HomeDir()
	if err != nil {
		log.Fatal(4, "Fail to get home directory: %v", err)
	}
	return home
}

func init() {
	var err error

	if appPath, err = exePath(); err != nil {
		log.Fatal(4, "fail to get app path: %v\n", err)
	}
	appPath = strings.Replace(appPath, "\\", "/", -1)

	// Determine and create .ssh path.
	SshPath = filepath.Join(homeDir(), ".ssh")
	if err = os.MkdirAll(SshPath, 0700); err != nil {
		log.Fatal(4, "fail to create SshPath(%s): %v\n", SshPath, err)
	}
}

// PublicKey represents a SSH key.
type PublicKey struct {
	Id                int64
	OwnerId           int64  `xorm:"UNIQUE(s) INDEX NOT NULL"`
	Name              string `xorm:"UNIQUE(s) NOT NULL"`
	Fingerprint       string
	Content           string    `xorm:"TEXT NOT NULL"`
	Created           time.Time `xorm:"CREATED"`
	Updated           time.Time
	HasRecentActivity bool `xorm:"-"`
	HasUsed           bool `xorm:"-"`
}

// GetAuthorizedString generates and returns formatted public key string for authorized_keys file.
func (key *PublicKey) GetAuthorizedString() string {
	return fmt.Sprintf(_TPL_PUBLICK_KEY, appPath, key.Id, key.Content)
}

var (
	MinimumKeySize = map[string]int{
		"(ED25519)": 256,
		"(ECDSA)":   256,
		"(NTRU)":    1087,
		"(MCE)":     1702,
		"(McE)":     1702,
		"(RSA)":     2048,
		"(DSA)":     1024,
	}
)

// CheckPublicKeyString checks if the given public key string is recognized by SSH.
func CheckPublicKeyString(content string) (bool, error) {
	if strings.ContainsAny(content, "\n\r") {
		return false, errors.New("Only a single line with a single key please")
	}

	// write the key to a file…
	tmpFile, err := ioutil.TempFile(os.TempDir(), "keytest")
	if err != nil {
		return false, err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	tmpFile.WriteString(content)
	tmpFile.Close()

	// Check if ssh-keygen recognizes its contents.
	stdout, stderr, err := process.Exec("CheckPublicKeyString", "ssh-keygen", "-l", "-f", tmpPath)
	if err != nil {
		return false, errors.New("ssh-keygen -l -f: " + stderr)
	} else if len(stdout) < 2 {
		return false, errors.New("ssh-keygen returned not enough output to evaluate the key")
	}

	// The ssh-keygen in Windows does not print key type, so no need go further.
	if setting.IsWindows {
		return true, nil
	}

	sshKeygenOutput := strings.Split(stdout, " ")
	if len(sshKeygenOutput) < 4 {
		return false, errors.New("Not enough fields returned by ssh-keygen -l -f")
	}

	// Check if key type and key size match.
	keySize, err := com.StrTo(sshKeygenOutput[0]).Int()
	if err != nil {
		return false, errors.New("Cannot get key size of the given key")
	}
	keyType := strings.TrimSpace(sshKeygenOutput[len(sshKeygenOutput)-1])
	if minimumKeySize := MinimumKeySize[keyType]; minimumKeySize == 0 {
		return false, errors.New("Sorry, unrecognized public key type")
	} else if keySize < minimumKeySize {
		return false, fmt.Errorf("The minimum accepted size of a public key %s is %d", keyType, minimumKeySize)
	}

	return true, nil
}

// saveAuthorizedKeyFile writes SSH key content to authorized_keys file.
func saveAuthorizedKeyFile(key *PublicKey) error {
	sshOpLocker.Lock()
	defer sshOpLocker.Unlock()

	fpath := filepath.Join(SshPath, "authorized_keys")
	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	finfo, err := f.Stat()
	if err != nil {
		return err
	}

	// FIXME: following command does not support in Windows.
	if !setting.IsWindows {
		if finfo.Mode().Perm() > 0600 {
			log.Error(4, "authorized_keys file has unusual permission flags: %s - setting to -rw-------", finfo.Mode().Perm().String())
			if err = f.Chmod(0600); err != nil {
				return err
			}
		}
	}

	_, err = f.WriteString(key.GetAuthorizedString())
	return err
}

// AddPublicKey adds new public key to database and authorized_keys file.
func AddPublicKey(key *PublicKey) (err error) {
	has, err := x.Get(key)
	if err != nil {
		return err
	} else if has {
		return ErrKeyAlreadyExist
	}

	// Calculate fingerprint.
	tmpPath := strings.Replace(path.Join(os.TempDir(), fmt.Sprintf("%d", time.Now().Nanosecond()),
		"id_rsa.pub"), "\\", "/", -1)
	os.MkdirAll(path.Dir(tmpPath), os.ModePerm)
	if err = ioutil.WriteFile(tmpPath, []byte(key.Content), os.ModePerm); err != nil {
		return err
	}
	stdout, stderr, err := process.Exec("AddPublicKey", "ssh-keygen", "-l", "-f", tmpPath)
	if err != nil {
		return errors.New("ssh-keygen -l -f: " + stderr)
	} else if len(stdout) < 2 {
		return errors.New("Not enough output for calculating fingerprint")
	}
	key.Fingerprint = strings.Split(stdout, " ")[1]

	// Save SSH key.
	if _, err = x.Insert(key); err != nil {
		return err
	} else if err = saveAuthorizedKeyFile(key); err != nil {
		// Roll back.
		if _, err2 := x.Delete(key); err2 != nil {
			return err2
		}
		return err
	}

	return nil
}

// GetPublicKeyById returns public key by given ID.
func GetPublicKeyById(keyId int64) (*PublicKey, error) {
	key := new(PublicKey)
	has, err := x.Id(keyId).Get(key)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrKeyNotExist
	}
	return key, nil
}

// ListPublicKey returns a list of all public keys that user has.
func ListPublicKey(uid int64) ([]*PublicKey, error) {
	keys := make([]*PublicKey, 0, 5)
	err := x.Find(&keys, &PublicKey{OwnerId: uid})
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		key.HasUsed = key.Updated.After(key.Created)
		key.HasRecentActivity = key.Updated.Add(7 * 24 * time.Hour).After(time.Now())
	}
	return keys, nil
}

// rewriteAuthorizedKeys finds and deletes corresponding line in authorized_keys file.
func rewriteAuthorizedKeys(key *PublicKey, p, tmpP string) error {
	sshOpLocker.Lock()
	defer sshOpLocker.Unlock()

	fr, err := os.Open(p)
	if err != nil {
		return err
	}
	defer fr.Close()

	fw, err := os.OpenFile(tmpP, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer fw.Close()

	isFound := false
	keyword := fmt.Sprintf("key-%d", key.Id)
	buf := bufio.NewReader(fr)
	for {
		line, errRead := buf.ReadString('\n')
		line = strings.TrimSpace(line)

		if errRead != nil {
			if errRead != io.EOF {
				return errRead
			}

			// Reached end of file, if nothing to read then break,
			// otherwise handle the last line.
			if len(line) == 0 {
				break
			}
		}

		// Found the line and copy rest of file.
		if !isFound && strings.Contains(line, keyword) && strings.Contains(line, key.Content) {
			isFound = true
			continue
		}
		// Still finding the line, copy the line that currently read.
		if _, err = fw.WriteString(line + "\n"); err != nil {
			return err
		}

		if errRead == io.EOF {
			break
		}
	}
	return nil
}

// UpdatePublicKey updates given public key.
func UpdatePublicKey(key *PublicKey) error {
	_, err := x.Id(key.Id).AllCols().Update(key)
	return err
}

// DeletePublicKey deletes SSH key information both in database and authorized_keys file.
func DeletePublicKey(key *PublicKey) error {
	has, err := x.Get(key)
	if err != nil {
		return err
	} else if !has {
		return ErrKeyNotExist
	}

	if _, err = x.Delete(key); err != nil {
		return err
	}

	fpath := filepath.Join(SshPath, "authorized_keys")
	tmpPath := filepath.Join(SshPath, "authorized_keys.tmp")
	if err = rewriteAuthorizedKeys(key, fpath, tmpPath); err != nil {
		return err
	} else if err = os.Remove(fpath); err != nil {
		return err
	}
	return os.Rename(tmpPath, fpath)
}
