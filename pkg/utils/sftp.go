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
saves them into the local directory, then deletes them from the server (to satisfy Amazon’s receiving test).
Returns a slice of paths to the downloaded files.

Parameters:
  - host:           SFTP server hostname or IP.
  - port:           SFTP port (usually 22).
  - username:       Username for SSH authentication.
  - privateKeyPath: Path to the SSH private key file for key-based auth.
  - remoteDir:      Directory on the SFTP server containing files to download (e.g. "download").
  - localDir:       Local base directory where files will be saved.

Returns:
  - []string: List of local file paths downloaded.
  - error:    Non-nil if any step fails.
*/
func FetchFilesOverSFTP(host string, port int, username, privateKeyPath, remoteDir, localDir string) ([]string, error) {
	// Load private key
	key, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// SSH client config
	sshCfg := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := ssh.Dial("tcp", addr, sshCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to dial SSH: %w", err)
	}
	defer conn.Close()

	sftpClient, err := sftp.NewClient(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// List entries in remoteDir
	entries, err := sftpClient.ReadDir(remoteDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read remote directory %s: %w", remoteDir, err)
	}

	// Ensure local directory exists
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create local directory %s: %w", localDir, err)
	}

	var downloaded []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		remotePath := path.Join(remoteDir, entry.Name())
		localPath := filepath.Join(localDir, entry.Name())

		// Open remote file
		rf, err := sftpClient.Open(remotePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open remote file %s: %w", remotePath, err)
		}

		// Create local file
		lf, err := os.Create(localPath)
		if err != nil {
			rf.Close()
			return nil, fmt.Errorf("failed to create local file %s: %w", localPath, err)
		}

		// Copy contents
		if _, err := io.Copy(lf, rf); err != nil {
			lf.Close()
			rf.Close()
			return nil, fmt.Errorf("failed to copy %s to %s: %w", remotePath, localPath, err)
		}

		lf.Close()
		rf.Close()

		// Delete remote file so Amazon’s receiving test passes
		if err := sftpClient.Remove(remotePath); err != nil {
			return nil, fmt.Errorf("failed to delete remote file %s: %w", remotePath, err)
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
	// load private key
	key, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return fmt.Errorf("read private key: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}

	// SSH config
	sshCfg := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	// dial
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), sshCfg)
	if err != nil {
		return fmt.Errorf("ssh dial: %w", err)
	}
	defer conn.Close()

	// sftp client
	client, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("sftp client: %w", err)
	}
	defer client.Close()

	// ensure remoteDir is relative, then prepend "/"
	remoteDir = strings.TrimPrefix(remoteDir, "/")
	remotePath := path.Join("/", remoteDir, fileName)

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
