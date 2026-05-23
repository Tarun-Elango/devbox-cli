package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"devbox-cli/internal/api"
	"devbox-cli/internal/config"
)

// TestMode disables real API calls; each command prints a stub message instead.
var TestMode bool

// Login prompts for credentials, POSTs to /v1/auth/login, and saves the returned token.
func Login(args []string) {
	if TestMode {
		fmt.Println("[test] login: done")
		return
	}
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading username: %v\n", err)
		os.Exit(1)
	}
	username = strings.TrimSpace(username)

	password, err := readPassword("Password: ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading password: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	client := api.New(cfg.ServerURL, "")

	resp, err := client.Post("/v1/auth/login", map[string]string{
		"username": username,
		"password": password,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "login failed: %v\n", err)
		os.Exit(1)
	}
	if err := api.CheckStatus(resp); err != nil {
		fmt.Fprintf(os.Stderr, "login failed: %v\n", err)
		os.Exit(1)
	}

	var result struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refreshToken"`
	}
	if err := api.DecodeJSON(resp, &result); err != nil {
		fmt.Fprintf(os.Stderr, "login failed: %v\n", err)
		os.Exit(1)
	}
	if result.Token == "" {
		fmt.Fprintln(os.Stderr, "login failed: server did not return a token")
		os.Exit(1)
	}

	cfg.Token = result.Token
	cfg.RefreshToken = result.RefreshToken
	cfg.TokenExpiry = config.ParseTokenExpiry(result.Token)
	if err := config.Save(cfg); err != nil { // writes to ~/.devbox/config.json
		fmt.Fprintf(os.Stderr, "save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Logged in successfully.")
}

// Signup prompts for credentials, POSTs to /v1/auth/signup, and saves the returned token.
func Signup(args []string) {
	if TestMode {
		fmt.Println("[test] signup: done")
		return
	}
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading username: %v\n", err)
		os.Exit(1)
	}
	username = strings.TrimSpace(username)

	password, err := readPassword("Password: ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading password: %v\n", err)
		os.Exit(1)
	}

	confirm, err := readPassword("Confirm password: ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading password confirmation: %v\n", err)
		os.Exit(1)
	}
	if len(password) < 8 {
		fmt.Fprintln(os.Stderr, "signup failed: password must be at least 8 characters")
		os.Exit(1)
	}
	if password != confirm {
		fmt.Fprintln(os.Stderr, "signup failed: passwords do not match")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	client := api.New(cfg.ServerURL, "")

	resp, err := client.Post("/v1/auth/signup", map[string]string{
		"username": username,
		"password": password,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "signup failed: %v\n", err)
		os.Exit(1)
	}
	if err := api.CheckStatus(resp); err != nil {
		fmt.Fprintf(os.Stderr, "signup failed: %v\n", err)
		os.Exit(1)
	}

	var result struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refreshToken"`
	}
	if err := api.DecodeJSON(resp, &result); err != nil {
		fmt.Fprintf(os.Stderr, "signup failed: %v\n", err)
		os.Exit(1)
	}
	if result.Token == "" {
		fmt.Fprintln(os.Stderr, "signup failed: server did not return a token")
		os.Exit(1)
	}

	cfg.Token = result.Token
	cfg.RefreshToken = result.RefreshToken
	cfg.TokenExpiry = config.ParseTokenExpiry(result.Token)
	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Account created and logged in.")
}

// Logout clears the locally stored JWT. Because tokens are stateless there is
// no server-side session to invalidate — removing the local token is sufficient.
func Logout() {
	if TestMode {
		fmt.Println("[test] logout: done")
		return
	}

	// just need to clear the saved tokens; there is no server-side session to invalidate
	if err := config.Clear(); err != nil {
		fmt.Fprintf(os.Stderr, "clear config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Logged out.")
}

// readPassword prints prompt, disables terminal echo, reads a line, then
// restores echo. Falls back to plain stdin read if the terminal cannot be
// configured (e.g. piped input).
func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)

	fd := int(os.Stdin.Fd())

	// Try to disable echo via termios. If stdin is not a real terminal this
	// syscall will fail and we fall through to a plain read.
	var oldState syscall.Termios
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(fd), ioctlReadTermios, uintptr(unsafe.Pointer(&oldState))); errno == 0 {

		newState := oldState
		newState.Lflag &^= syscall.ECHO
		newState.Lflag |= syscall.ICANON | syscall.ISIG
		syscall.Syscall(syscall.SYS_IOCTL,
			uintptr(fd), ioctlWriteTermios, uintptr(unsafe.Pointer(&newState)))

		defer func() {
			syscall.Syscall(syscall.SYS_IOCTL,
				uintptr(fd), ioctlWriteTermios, uintptr(unsafe.Pointer(&oldState)))
			fmt.Println() // print the newline the suppressed echo swallowed
		}()
	}

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	return strings.TrimSpace(line), err
}
