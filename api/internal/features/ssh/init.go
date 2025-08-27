package ssh

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/melbahja/goph"
	"github.com/raghavyuva/nixopus-api/internal/config"
	"github.com/raghavyuva/nixopus-api/internal/types"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

type SSH struct {
	PrivateKey          string `json:"private_key"`
	PublicKey           string `json:"public_key"`
	Host                string `json:"host"`
	User                string `json:"user"`
	Port                uint   `json:"port"`
	Password            string `json:"password"`
	PrivateKeyProtected string `json:"private_key_protected"`
}

func NewSSH() *SSH {
	return &SSH{
		PrivateKey:          config.AppConfig.SSH.PrivateKey,
		Host:                config.AppConfig.SSH.Host,
		User:                config.AppConfig.SSH.User,
		Port:                config.AppConfig.SSH.Port,
		Password:            config.AppConfig.SSH.Password,
		PrivateKeyProtected: config.AppConfig.SSH.PrivateKeyProtected,
	}
}

// NewSSHWithContext creates SSH client that's aware of request context
// If a server is found in context, it uses that server's SSH config
// Otherwise falls back to default config
func NewSSHWithContext(ctx context.Context) *SSH {
	// Check if there's a server in the context
	if server := getServerFromContext(ctx); server != nil {
		log.Printf("Server found in context: %v\n", server)
		return &SSH{
			PrivateKey: getStringValue(server.SSHPrivateKeyPath),
			Host:       server.Host,
			User:       server.Username,
			Port:       uint(server.Port),
			Password:   getStringValue(server.SSHPassword),
		}
	}
	log.Printf("No server found in context, using default config")
	// Fallback to default config
	return NewSSH()
}

// NewSSHWithServer creates SSH client with specific server configuration
// If a server is provided, it uses that server's SSH config
// Otherwise falls back to default config
func NewSSHWithServer(server *types.Server) *SSH {
	// Check if server is provided
	if server != nil {
		log.Printf("Using server SSH config: %s@%s:%d", server.Username, server.Host, server.Port)
		return &SSH{
			PrivateKey: getStringValue(server.SSHPrivateKeyPath),
			Host:       server.Host,
			User:       server.Username,
			Port:       uint(server.Port),
			Password:   getStringValue(server.SSHPassword),
		}
	}
	log.Printf("No server provided, using default SSH config")
	// Fallback to default config
	return NewSSH()
}

// Helper function to safely get string value from pointer
func getStringValue(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

// Helper function to extract server from context
func getServerFromContext(ctx context.Context) *types.Server {
	if server := ctx.Value(types.ServerIDKey); server != nil {
		if s, ok := server.(*types.Server); ok {
			return s
		}
	}
	return nil
}

// NewSSHFromConfig creates SSH client from custom config (for server-specific connections)
func NewSSHFromConfig(sshConfig *types.SSHConfig) *SSH {
	return &SSH{
		PrivateKey:          sshConfig.PrivateKey,
		Host:                sshConfig.Host,
		User:                sshConfig.User,
		Port:                sshConfig.Port,
		Password:            sshConfig.Password,
		PrivateKeyProtected: sshConfig.PrivateKeyProtected,
	}
}

func (s *SSH) ConnectWithPassword() (*goph.Client, error) {
	if s.Password == "" {
		return nil, fmt.Errorf("password is required for SSH connection")
	}

	auth := goph.Password(s.Password)

	client, err := goph.NewConn(&goph.Config{
		User:     s.User,
		Addr:     s.Host,
		Port:     uint(s.Port),
		Auth:     auth,
		Callback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to establish SSH connection with password: %w", err)
	}

	return client, nil
}

func (s *SSH) Connect() (*goph.Client, error) {
	if s.User == "" || s.Host == "" {
		return nil, fmt.Errorf("user and host are required for SSH connection")
	}

	client, err := s.ConnectWithPrivateKey()
	if err == nil {
		return client, nil
	}

	fmt.Printf("private key connection failed: %v\n", err)

	client, err = s.ConnectWithPassword()
	if err != nil {
		return nil, fmt.Errorf("failed to connect with both private key and password: %w", err)
	}

	return client, nil
}

func (s *SSH) ConnectWithPrivateKey() (*goph.Client, error) {
	if s.PrivateKey == "" {
		return nil, fmt.Errorf("private key is required for SSH connection")
	}

	auth, err := goph.Key(s.PrivateKey, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH auth from private key: %w", err)
	}

	client, err := goph.NewConn(&goph.Config{
		User:     s.User,
		Addr:     s.Host,
		Port:     uint(s.Port),
		Auth:     auth,
		Callback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to establish SSH connection with private key: %w", err)
	}

	return client, nil
}

// func (s *SSH) ConnectWithPrivateKeyProtected() (*goph.Client, error) {
// 	auth, err := goph.Key(s.PrivateKeyProtected, "")

// 	if err != nil {
// 		log.Fatalf("SSH connection failed: %v", err)
// 	}

// 	client, err := goph.NewConn(&goph.Config{
// 		User:     s.User,
// 		Addr:     s.Host,
// 		Port:     uint(s.Port),
// 		Auth:     auth,
// 		Callback: ssh.InsecureIgnoreHostKey(),
// 	})
// 	if err != nil {
// 		log.Fatalf("SSH connection failed: %v", err)
// 	}

// 	defer client.Close()
// 	return client, nil
// }

func (s *SSH) RunCommand(cmd string) (string, error) {
	client, err := s.Connect()
	if err != nil {
		return "", err
	}
	output, err := client.Run(cmd)

	if err != nil {
		return "", err
	}

	return string(output), nil
}

func (s *SSH) Terminal() {
	client, err := s.Connect()
	if err != nil {
		fmt.Print("Failed to connect to ssh")
		return
	}
	session, err := client.NewSession()
	if err != nil {
		fmt.Printf("Failed to create session: %s\n", err)
		return
	}
	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	fileDescriptor := int(os.Stdin.Fd())
	if terminal.IsTerminal(fileDescriptor) {
		originalState, err := terminal.MakeRaw(fileDescriptor)
		if err != nil {
			panic(err)
		}
		defer terminal.Restore(fileDescriptor, originalState)

		termWidth, termHeight, err := terminal.GetSize(fileDescriptor)
		if err != nil {
			panic(err)
		}

		err = session.RequestPty("xterm-256color", termHeight, termWidth, modes)
		if err != nil {
			panic(err)
		}
	}

	err = session.Shell()
	if err != nil {
		return
	}
	session.Wait()
}
