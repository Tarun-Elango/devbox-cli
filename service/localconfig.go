package service

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

var errSSHHostNotFound = errors.New("ssh host not found")

func sshHostNotFound(host string) error {
	return fmt.Errorf("host %q does not exist: %w", host, errSSHHostNotFound)
}

const sshConfigUpdateAttempts = 3

func sshConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ssh", "config"), nil // should be ~/.ssh/config
}

func readSSHConfig() (string, error) {
	path, err := sshConfigPath() // get the path to the ssh config file
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path) // read the ssh config file
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func writeSSHConfig(content string) error {
	path, err := sshConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0600) // write the ssh config file
}

func updateSSHConfig(update func(content string) (string, error)) error {
	content, err := readSSHConfig()
	if err != nil {
		return err
	}
	updated, err := update(content)
	if err != nil {
		return err
	}
	if updated == content {
		return nil
	}
	return writeSSHConfig(updated)
}

func updateSSHConfigWithRetry(update func(content string) (string, error)) error {
	var err error
	for attempt := 0; attempt < sshConfigUpdateAttempts; attempt++ {
		err = updateSSHConfig(update)
		if err == nil {
			return nil
		}
	}
	return err
}

type hostBlock struct {
	start int
	end   int
	hosts []string
}

func sshConfigLines(content string) (lines []string, hadTrailingNewline bool) {
	if content == "" {
		return nil, false
	}
	hadTrailingNewline = strings.HasSuffix(content, "\n")
	lines = strings.Split(content, "\n")
	if hadTrailingNewline && len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines, hadTrailingNewline
}

func formatSSHConfig(lines []string, hadTrailingNewline bool) string {
	result := strings.Join(lines, "\n")
	if hadTrailingNewline {
		result += "\n"
	}
	return result
}

func parseHostBlocks(content string) []hostBlock {
	lines, _ := sshConfigLines(content)
	var blocks []hostBlock
	var current *hostBlock

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "host ") {
			if current != nil {
				current.end = i
				blocks = append(blocks, *current)
			}
			parts := strings.Fields(trimmed)
			hosts := []string{}
			if len(parts) > 1 {
				hosts = parts[1:]
			}
			current = &hostBlock{start: i, end: len(lines), hosts: hosts}
		}
	}
	if current != nil {
		blocks = append(blocks, *current)
	}
	return blocks
}

func hostInBlock(block hostBlock, name string) bool {
	for _, h := range block.hosts {
		if h == name {
			return true
		}
	}
	return false
}

func OutpostHostName(name string) string {
	return "outpost-" + name
}

func findBlockByHost(content, name string) (hostBlock, bool) {
	// for each block in the ssh config file, check if the host is in the block
	for _, block := range parseHostBlocks(content) {
		if hostInBlock(block, name) {
			return block, true
		}
	}
	return hostBlock{}, false
}

func formatHostBlock(name, ipAddress string) string {
	return fmt.Sprintf(
		"Host %s\n    HostName %s\n    User ec2-user\n    IdentityFile ~/.ssh/id_ed25519\n    StrictHostKeyChecking accept-new\n",
		name, ipAddress,
	)
}

func validateSSHBoxName(name string) error {
	if name == "" {
		return fmt.Errorf("host name cannot be empty")
	}
	if strings.ContainsAny(name, " \t\n\r") {
		return fmt.Errorf("host name %q contains invalid characters", name)
	}
	return nil
}

func validateSSHIPAddress(ipAddress string) error {
	if ipAddress == "" {
		return fmt.Errorf("ip address cannot be empty")
	}
	if strings.ContainsAny(ipAddress, " \t\n\r") {
		return fmt.Errorf("ip address %q contains invalid characters", ipAddress)
	}
	if net.ParseIP(ipAddress) == nil {
		return fmt.Errorf("ip address %q is not a valid IP address", ipAddress)
	}
	return nil
}

// add new host in .ssh/config ( name, ip address)
func AddHost(name, ipAddress string) error {
	if err := validateSSHBoxName(name); err != nil {
		return err
	}
	if err := validateSSHIPAddress(ipAddress); err != nil {
		return err
	}
	host := OutpostHostName(name)
	return updateSSHConfig(func(content string) (string, error) {
		if _, found := findBlockByHost(content, host); found { // check if the host already exists
			return "", fmt.Errorf("host %q already exists", host)
		}
		if content != "" && !strings.HasSuffix(content, "\n") { // if empty or not a new line, add a new line
			content += "\n"
		}
		content += formatHostBlock(host, ipAddress)
		return content, nil
	})
}

// update ip address in .ssh/config (name, ip address), for the given host with name, update the ip address
func UpdateHost(name, ipAddress string) error {
	if err := validateSSHBoxName(name); err != nil {
		return err
	}
	if err := validateSSHIPAddress(ipAddress); err != nil {
		return err
	}
	host := OutpostHostName(name)
	return updateSSHConfig(func(content string) (string, error) {
		block, found := findBlockByHost(content, host)
		if !found {
			return "", sshHostNotFound(host)
		}

		lines, trailing := sshConfigLines(content)
		blockLines := lines[block.start:block.end]

		hostnameUpdated := false
		for i, line := range blockLines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(strings.ToLower(trimmed), "hostname ") {
				indent := lineIndent(line)
				blockLines[i] = indent + "HostName " + ipAddress
				hostnameUpdated = true
				break
			}
		}
		if !hostnameUpdated {
			insert := blockOptionIndent(blockLines) + "HostName " + ipAddress
			updatedBlock := make([]string, 0, len(blockLines)+1)
			updatedBlock = append(updatedBlock, blockLines[0])
			updatedBlock = append(updatedBlock, insert)
			updatedBlock = append(updatedBlock, blockLines[1:]...)
			blockLines = updatedBlock
		}

		newLines := append(lines[:block.start], append(blockLines, lines[block.end:]...)...)
		return formatSSHConfig(newLines, trailing), nil
	})
}

// RenameHost rewrites an existing outpost host alias while preserving its SSH options.
func RenameHost(oldName, newName string) error {
	if err := validateSSHBoxName(oldName); err != nil {
		return err
	}
	if err := validateSSHBoxName(newName); err != nil {
		return err
	}
	oldHost := OutpostHostName(oldName)
	newHost := OutpostHostName(newName)
	if oldHost == newHost {
		return nil
	}

	return updateSSHConfigWithRetry(func(content string) (string, error) {
		block, found := findBlockByHost(content, oldHost)
		if !found {
			return "", sshHostNotFound(oldHost)
		}
		if _, found := findBlockByHost(content, newHost); found {
			return "", fmt.Errorf("host %q already exists", newHost)
		}

		lines, trailing := sshConfigLines(content)
		blockLines := lines[block.start:block.end]
		if len(blockLines) == 0 {
			return "", fmt.Errorf("host %q block is empty", oldHost)
		}

		hosts := append([]string(nil), block.hosts...)
		for i, host := range hosts {
			if host == oldHost {
				hosts[i] = newHost
				break
			}
		}
		blockLines[0] = lineIndent(blockLines[0]) + "Host " + strings.Join(hosts, " ")

		newLines := append(lines[:block.start], append(blockLines, lines[block.end:]...)...)
		return formatSSHConfig(newLines, trailing), nil
	})
}

// syncSSHHostIP updates the HostName for an existing entry, or adds one if missing.
func syncSSHHostIP(name, ipAddress string) error {
	if ipAddress == "" {
		return nil
	}
	if err := UpdateHost(name, ipAddress); err != nil {
		if !errors.Is(err, errSSHHostNotFound) {
			return err
		}
		return AddHost(name, ipAddress)
	}
	return nil
}

const forwardAgentOption = "ForwardAgent yes"

func isForwardAgentLine(line string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "forwardagent")
}

// forwardAgentEnabledInBlock reports whether the first ForwardAgent directive
// in blockLines enables agent forwarding. OpenSSH uses the first match.
func forwardAgentEnabledInBlock(blockLines []string) bool {
	for _, line := range blockLines {
		if !isForwardAgentLine(line) {
			continue
		}
		parts := strings.Fields(strings.TrimSpace(line))
		if len(parts) >= 2 {
			return strings.EqualFold(parts[1], "yes")
		}
		return false
	}
	return false
}

func countForwardAgentLines(blockLines []string) int {
	n := 0
	for _, line := range blockLines {
		if isForwardAgentLine(line) {
			n++
		}
	}
	return n
}

func removeForwardAgentLines(blockLines []string) []string {
	updated := make([]string, 0, len(blockLines))
	for _, line := range blockLines {
		if !isForwardAgentLine(line) {
			updated = append(updated, line)
		}
	}
	return updated
}

func lineIndent(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

// blockOptionIndent returns the indentation used by a block's existing
// options, falling back to a 4-space default for blocks with none.
func blockOptionIndent(blockLines []string) string {
	for _, line := range blockLines {
		if trimmed := strings.TrimSpace(line); trimmed != "" && !strings.HasPrefix(strings.ToLower(trimmed), "host ") {
			return lineIndent(line)
		}
	}
	return "    "
}

// appendBlockOption inserts option as the last option in the block, i.e.
// after the last non-blank line but before any trailing blank separator
// lines that belong to the block.
func appendBlockOption(blockLines []string, option string) []string {
	end := len(blockLines)
	for end > 0 && strings.TrimSpace(blockLines[end-1]) == "" {
		end--
	}
	updated := make([]string, 0, len(blockLines)+1)
	updated = append(updated, blockLines[:end]...)
	updated = append(updated, blockOptionIndent(blockLines)+option)
	updated = append(updated, blockLines[end:]...)
	return updated
}

// ForwardAgentEnabled reports whether ForwardAgent yes is set for a outpost host.
// read ssh config, get block , and check if ForwardAgent yes is set
func ForwardAgentEnabled(name string) (bool, error) {
	if err := validateSSHBoxName(name); err != nil {
		return false, err
	}
	content, err := readSSHConfig()
	if err != nil {
		return false, err
	}
	block, found := findBlockByHost(content, OutpostHostName(name))
	if !found {
		return false, sshHostNotFound(OutpostHostName(name))
	}
	lines, _ := sshConfigLines(content)
	return forwardAgentEnabledInBlock(lines[block.start:block.end]), nil
}

// EnableForwardAgent adds ForwardAgent yes to a outpost host block when missing.
func EnableForwardAgent(name string) error {
	if err := validateSSHBoxName(name); err != nil {
		return err
	}
	host := OutpostHostName(name)
	return updateSSHConfigWithRetry(func(content string) (string, error) {
		block, found := findBlockByHost(content, host)
		if !found {
			return "", sshHostNotFound(host)
		}

		lines, trailing := sshConfigLines(content)
		blockLines := lines[block.start:block.end]
		if forwardAgentEnabledInBlock(blockLines) && countForwardAgentLines(blockLines) == 1 {
			return content, nil
		}
		updatedBlock := appendBlockOption(removeForwardAgentLines(blockLines), forwardAgentOption)
		newLines := append(lines[:block.start], append(updatedBlock, lines[block.end:]...)...)
		return formatSSHConfig(newLines, trailing), nil
	})
}

// DisableForwardAgent removes ForwardAgent yes from a outpost host block when present.
func DisableForwardAgent(name string) error {
	if err := validateSSHBoxName(name); err != nil {
		return err
	}
	host := OutpostHostName(name)
	return updateSSHConfigWithRetry(func(content string) (string, error) {
		block, found := findBlockByHost(content, host)
		if !found {
			return "", sshHostNotFound(host)
		}

		lines, trailing := sshConfigLines(content)
		blockLines := lines[block.start:block.end]
		if countForwardAgentLines(blockLines) == 0 {
			return content, nil
		}

		updatedBlock := removeForwardAgentLines(blockLines)
		newLines := append(lines[:block.start], append(updatedBlock, lines[block.end:]...)...)
		return formatSSHConfig(newLines, trailing), nil
	})
}

// delete host (name)
func DeleteHost(name string) error {
	if err := validateSSHBoxName(name); err != nil {
		return err
	}
	host := OutpostHostName(name)
	return updateSSHConfig(func(content string) (string, error) {
		block, found := findBlockByHost(content, host)
		if !found {
			return content, nil // if the host does not exist, do nothing
		}

		lines, trailing := sshConfigLines(content)
		newLines := append(lines[:block.start], lines[block.end:]...)
		return formatSSHConfig(newLines, trailing), nil
	})
}
