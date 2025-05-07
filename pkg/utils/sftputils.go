// pkg/utils/sftputils.go
package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
	"github.com/pkg/sftp"
)

/*
FetchFilesOverSFTP connects to an SFTP server, downloads all files in the specified remote directory,
and saves them into the local directory. Returns a slice of paths to the downloaded files.

Parameters:
  - host:           SFTP server hostname or IP.
  - port:           SFTP port (usually 22).
  - username:       Username for SSH authentication.
  - privateKeyPath: Path to the SSH private key file for key-based auth.
  - remoteDir:      Directory on the SFTP server containing files to download.
  - localDir:       Local base directory where files will be saved.

Returns:
  - []string: List of local file paths downloaded.
  - error:    Non-nil if any step fails.
*/
func FetchFilesOverSFTP(host string, port int, username, privateKeyPath, remoteDir, localDir string) ([]string, error) {
	key, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout: 10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := ssh.Dial("tcp", addr, config)
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

	var downloaded []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		remotePath := filepath.Join(remoteDir, entry.Name())

		// ensure local directory exists
		if err := os.MkdirAll(localDir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create local directory %s: %w", localDir, err)
		}

		// open remote file
		rf, err := client.Open(remotePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open remote file %s: %w", remotePath, err)
		}
		defer rf.Close()

		// create local file
		localPath := filepath.Join(localDir, entry.Name())
		lf, err := os.Create(localPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create local file %s: %w", localPath, err)
		}
		defer lf.Close()

		// copy contents
		if _, err := io.Copy(lf, rf); err != nil {
			return nil, fmt.Errorf("failed to copy %s to %s: %w", remotePath, localPath, err)
		}

		downloaded = append(downloaded, localPath)
	}

	return downloaded, nil
}
