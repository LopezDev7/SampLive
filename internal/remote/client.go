package remote

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"samplive/internal/config"
)

// Transport is the generic remote interface. Future transports (e.g. a
// Pterodactyl panel API) implement the same contract.
type Transport interface {
	Upload(localPath, remotePath string) error
	Exec(command string) (string, error)
	Close() error
}

// Client is an SFTP/SSH transport.
type Client struct {
	cfg  *config.Remote
	conn *ssh.Client
}

// Connect opens an SSH connection to the configured remote host.
func Connect(cfg *config.Remote) (*Client, error) {
	methods, err := authMethods(cfg)
	if err != nil {
		return nil, err
	}
	port := cfg.SSHPort
	if port == 0 {
		port = 22
	}
	hostKey, err := hostKeyCallback(cfg, "")
	if err != nil {
		return nil, err
	}
	clientCfg := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            methods,
		HostKeyCallback: hostKey,
		Timeout:         10 * time.Second,
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(port))
	conn, err := ssh.Dial("tcp", addr, clientCfg)
	if err != nil {
		return nil, fmt.Errorf("remote: ssh dial %s: %w", addr, err)
	}
	return &Client{cfg: cfg, conn: conn}, nil
}

// hostKeyCallback returns the callback used to verify the remote host key.
// By default it checks the user's ~/.ssh/known_hosts file. Passing
// knownHostsPath overrides the location (used by tests). Set
// remote.insecure_skip_host_key to skip the check entirely, which you
// usually don't want but is there for hosts with no home dir.
func hostKeyCallback(cfg *config.Remote, knownHostsPath string) (ssh.HostKeyCallback, error) {
	if cfg.InsecureSkipHostKey {
		return ssh.InsecureIgnoreHostKey(), nil
	}
	if knownHostsPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("remote: no home dir to find known_hosts: %w (set remote.insecure_skip_host_key: true to skip the check)", err)
		}
		knownHostsPath = filepath.Join(home, ".ssh", "known_hosts")
	}
	if _, err := os.Stat(knownHostsPath); err != nil {
		return nil, fmt.Errorf("remote: %s not found. add the server key with: ssh-keyscan %s >> %s (or set remote.insecure_skip_host_key: true to skip)", knownHostsPath, cfg.Host, knownHostsPath)
	}
	cb, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("remote: read %s: %w", knownHostsPath, err)
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if err := cb(hostname, remote, key); err != nil {
			return fmt.Errorf("remote: host key for %s not in %s. add it with: ssh-keyscan %s >> %s (or set remote.insecure_skip_host_key: true to skip)", cfg.Host, knownHostsPath, cfg.Host, knownHostsPath)
		}
		return nil
	}, nil
}

func authMethods(cfg *config.Remote) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod
	if cfg.Password != "" {
		methods = append(methods, ssh.Password(cfg.Password))
	}
	if cfg.Keyfile != "" {
		key, err := os.ReadFile(cfg.Keyfile)
		if err != nil {
			return nil, fmt.Errorf("remote: read keyfile %s: %w", cfg.Keyfile, err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("remote: parse keyfile %s: %w", cfg.Keyfile, err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}
	if len(methods) == 0 {
		return nil, fmt.Errorf("remote: no authentication method (set remote.password or remote.keyfile)")
	}
	return methods, nil
}

// Upload copies a local file to a remote path, creating parent directories.
func (c *Client) Upload(localPath, remotePath string) error {
	sftpC, err := sftp.NewClient(c.conn)
	if err != nil {
		return fmt.Errorf("remote: sftp: %w", err)
	}
	defer sftpC.Close()

	if dir := path.Dir(remotePath); dir != "." && dir != "/" {
		if err := sftpC.MkdirAll(dir); err != nil {
			return fmt.Errorf("remote: mkdir %s: %w", dir, err)
		}
	}
	src, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("remote: open %s: %w", localPath, err)
	}
	defer src.Close()

	dst, err := sftpC.Create(remotePath)
	if err != nil {
		return fmt.Errorf("remote: create %s: %w", remotePath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("remote: upload %s: %w", remotePath, err)
	}
	return nil
}

// Exec runs a command over SSH and returns its combined output.
func (c *Client) Exec(command string) (string, error) {
	sess, err := c.conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("remote: session: %w", err)
	}
	defer sess.Close()
	var out bytes.Buffer
	sess.Stdout = &out
	sess.Stderr = &out
	if err := sess.Run(command); err != nil {
		return strings.TrimSpace(out.String()), fmt.Errorf("remote: exec %q: %w", command, err)
	}
	return strings.TrimSpace(out.String()), nil
}

// Close closes the underlying SSH connection.
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
