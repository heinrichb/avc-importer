// pkg/utils/sftp.go
package utils

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

/*
FetchFilesOverSFTP connects to an SFTP server, downloads all files in the specified remote directory,
saves them into the local directory, then optionally deletes them from the server.
Returns a slice of paths to the downloaded files.

Parameters:
  - host:           SFTP server hostname or IP.
  - port:           SFTP port (usually 22).
  - username:       Username for SSH authentication.
  - privateKeyPath: Path to the SSH private key file for key-based auth.
  - remoteDir:      Directory on the SFTP server containing files to download (e.g. "download").
  - localDir:       Local base directory where files will be saved.
  - deleteAfter:    If true, remove each remote file after downloading.

Returns:
  - []string: List of local file paths downloaded.
  - error:    Non-nil if any step fails.
*/
func FetchFilesOverSFTP(
	host string,
	port int,
	username, privateKeyPath, remoteDir, localDir string,
	deleteAfter bool,
) ([]string, error) {
	// Amazon’s SFTP uses relative dirs under your home (e.g. "download"), so strip any leading slash.
	remoteDir = strings.TrimPrefix(remoteDir, "/")

	// Load private key
	key, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	sshCfg := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), sshCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to dial SSH: %w", err)
	}
	defer conn.Close()

	client, err := sftp.NewClient(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer client.Close()

	entries, err := client.ReadDir(remoteDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read remote directory %s: %w", remoteDir, err)
	}

	if len(entries) == 0 {
		fmt.Printf("No files found in %s\n", remoteDir)
		return nil, nil
	}

	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create local dir %s: %w", localDir, err)
	}

	var downloaded []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		remotePath := filepath.Join(remoteDir, entry.Name())
		localPath := filepath.Join(localDir, entry.Name())

		rf, err := client.Open(remotePath)
		if err != nil {
			return nil, fmt.Errorf("open remote %s: %w", remotePath, err)
		}
		lf, err := os.Create(localPath)
		if err != nil {
			rf.Close()
			return nil, fmt.Errorf("create local %s: %w", localPath, err)
		}
		if _, err := io.Copy(lf, rf); err != nil {
			rf.Close()
			lf.Close()
			return nil, fmt.Errorf("copy %s to %s: %w", remotePath, localPath, err)
		}
		rf.Close()
		lf.Close()

		if deleteAfter {
			if err := client.Remove(remotePath); err != nil {
				return nil, fmt.Errorf("delete remote %s: %w", remotePath, err)
			}
		}

		downloaded = append(downloaded, localPath)
	}

	return downloaded, nil
}

/*
UploadFileOverSFTP uploads the byte slice data as a file named fileName
into the remoteDir on the SFTP server.
Amazon’s SFTP uses “download” and “upload” as relative paths under your home directory,
so remoteDir should be provided without a leading “/”.

Parameters:
  - host:           SFTP server hostname.
  - port:           SFTP port (usually 22).
  - username:       SFTP username.
  - privateKeyPath: Path to your SSH private key.
  - remoteDir:      Directory on the SFTP server (e.g. "upload").
  - fileName:       Name of the file to create.
  - data:           Contents to write.
*/
func UploadFileOverSFTP(
	host string,
	port int,
	username, privateKeyPath, remoteDir, fileName string,
	data []byte,
) error {
	// Load private key
	key, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return fmt.Errorf("read private key: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}

	sshCfg := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), sshCfg)
	if err != nil {
		return fmt.Errorf("ssh dial: %w", err)
	}
	defer conn.Close()

	client, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("sftp client: %w", err)
	}
	defer client.Close()

	// Strip leading slash so remoteDir is relative under the SFTP home directory
	remoteDir = strings.TrimPrefix(remoteDir, "/")
	remotePath := path.Join(remoteDir, fileName)

	f, err := client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("create remote file %s: %w", remotePath, err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write remote file %s: %w", remotePath, err)
	}

	return nil
}
