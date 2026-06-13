package cmd

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"devbox-cli/scripts"
	"devbox-cli/service"
	"devbox-cli/service/localDb"
)

const idleStopUsage = "usage: devbox idle-stop <id> [in <minutes> | show | update <minutes> | delete]"

func IdleRouter(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, idleStopUsage)
		os.Exit(1)
	}

	id := args[0] // will be the aws instance id
	if id == "" {
		fmt.Fprintln(os.Stderr, "error: id is required")
		os.Exit(1)
	}
	switch args[1] {
	case "in":
		Idle(args)
	case "show":
		showIdleStop(args)
	case "update":
		updateIdleStop(args)
	case "delete":
		deleteIdleStop(args)
	default:
		fmt.Fprintf(os.Stderr, "idle-stop: unknown sub-command %q\n", args[1])
		fmt.Fprintln(os.Stderr, idleStopUsage)
		os.Exit(1)
	}

}

func Idle(args []string) {
	if TestMode {
		fmt.Println("[test] idle-stop: done")
		return
	}

	if len(args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: devbox idle-stop <id> in <minutes>")
		os.Exit(1)
	}

	id := args[0] // aws instance id
	if id == "" {
		fmt.Fprintln(os.Stderr, "error: id is required")
		os.Exit(1)
	}
	if args[1] != "in" {
		fmt.Fprintln(os.Stderr, "error: expected 'in' as second argument")
		os.Exit(1)
	}

	minutesInt, err := strconv.Atoi(args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: minutes must be an integer")
		os.Exit(1)
	}
	if minutesInt <= 0 {
		fmt.Fprintln(os.Stderr, "error: minutes must be greater than 0")
		os.Exit(1)
	}

	db, err := localDb.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	inst, err := db.GetInstanceByAwsInstanceIDAndUserID(id, service.LocalUserID)
	if err == sql.ErrNoRows {
		fmt.Fprintf(os.Stderr, "error: box not found: %s\n", id)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	sshStatus, err := service.GetSshStatus(id, service.LocalUserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if !sshStatus.Ready {
		fmt.Fprintln(os.Stderr, "error: box is not ready yet (EC2 status checks still pending)")
		os.Exit(1)
	}
	if sshStatus.Instance == nil {
		fmt.Fprintln(os.Stderr, "error: box is ready but instance details are unavailable, try again in a few minutes")
		os.Exit(1)
	}
	box := sshStatus.Instance
	if box.Status != "running" {
		fmt.Fprintf(os.Stderr, "error: box is %s, not running\n", box.Status)
		os.Exit(1)
	}
	host := box.IPAddress
	if host == "" {
		host = box.PrivateIPAddress
	}
	if host == "" {
		fmt.Fprintln(os.Stderr, "error: box has no IP address (is it running?)")
		os.Exit(1)
	}

	sshBin, err := exec.LookPath("ssh")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: ssh binary not found in PATH")
		os.Exit(1)
	}

	identity := defaultKeyPath()
	ready, err := checkDevboxReady(sshBin, identity, "ec2-user", host, "22")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: readiness probe failed: %v\n", err)
		os.Exit(1)
	}
	if !ready {
		fmt.Fprintln(os.Stderr, "error: devbox is not ready yet — try again in a minute")
		os.Exit(1)
	}

	// install idle stop to the host( ip address )
	if err := installIdleStop(sshBin, identity, "ec2-user", host, minutesInt); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// set the cell in table to the minutes for the specific instance
	if err := db.SetInstanceIdleStopMinutes(inst.AwsInstanceID, &minutesInt); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("idle-stop set to %d minutes for %s\n", minutesInt, inst.Name)
}

/*
mkdir -p /var/lib/devbox
Writes minutesInt to /var/lib/devbox/idle-stop-minutes
Writes current timestamp to /var/lib/devbox/last-activity
Installs check.bash → /usr/local/bin/devbox-idle-stop (chmod +x)
Installs the 3 systemd unit files under /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now devbox-idle-stop.timer
systemctl enable + systemctl start devbox-idle-stop-boot.service
*/
func installIdleStop(sshBin, identity, user, host string, minutes int) error {

	// covert the byte[] into base64 string
	checkB64 := base64.StdEncoding.EncodeToString(scripts.CheckBash)
	serviceB64 := base64.StdEncoding.EncodeToString(scripts.IdleStopService)
	timerB64 := base64.StdEncoding.EncodeToString(scripts.IdleStopTimer)
	bootB64 := base64.StdEncoding.EncodeToString(scripts.IdleStopBootService)

	script := fmt.Sprintf(`set -euo pipefail
mkdir -p /var/lib/devbox
echo %d > /var/lib/devbox/idle-stop-minutes
date +%%s > /var/lib/devbox/last-activity
echo %q | base64 -d > /usr/local/bin/devbox-idle-stop
chmod +x /usr/local/bin/devbox-idle-stop
echo %q | base64 -d > /etc/systemd/system/devbox-idle-stop.service
echo %q | base64 -d > /etc/systemd/system/devbox-idle-stop.timer
echo %q | base64 -d > /etc/systemd/system/devbox-idle-stop-boot.service
systemctl daemon-reload
systemctl enable --now devbox-idle-stop.timer
systemctl enable devbox-idle-stop-boot.service
systemctl start devbox-idle-stop-boot.service
`, minutes, checkB64, serviceB64, timerB64, bootB64)

	target := fmt.Sprintf("%s@%s", user, host)
	argv := append([]string{sshBin}, sshBaseArgs(identity, "22")...)
	argv = append(argv, target, "sudo", "bash", "-s")

	cmd := execCommand(sshBin, argv[1:]...)
	cmd.Stdin = strings.NewReader(script)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("idle-stop install failed: %s", msg)
	}
	return nil
}

// delete for a specific instance
/*
# 1. Stop and disable timer + boot service
sudo systemctl disable --now devbox-idle-stop.timer
sudo systemctl disable --now devbox-idle-stop-boot.service

# 2. Reload systemd
sudo systemctl daemon-reload

# 3. Remove systemd units (3 files now)
sudo rm -f /etc/systemd/system/devbox-idle-stop.timer
sudo rm -f /etc/systemd/system/devbox-idle-stop.service
sudo rm -f /etc/systemd/system/devbox-idle-stop-boot.service

# 4. Remove check script
sudo rm -f /usr/local/bin/devbox-idle-stop

# 5. Remove idle-stop config/state
sudo rm -f /var/lib/devbox/idle-stop-minutes
sudo rm -f /var/lib/devbox/last-activity

# 6. Reload + clear failed state
sudo systemctl daemon-reload
sudo systemctl reset-failed devbox-idle-stop.service 2>/dev/null || true
sudo systemctl reset-failed devbox-idle-stop-boot.service 2>/dev/null || true
*/
func deleteIdleStop(args []string) {
	if TestMode {
		fmt.Println("[test] idle-stop delete: done")
		return
	}

	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: devbox idle-stop <id> delete")
		os.Exit(1)
	}

	id := args[0]
	if id == "" {
		fmt.Fprintln(os.Stderr, "error: id is required")
		os.Exit(1)
	}
	if args[1] != "delete" {
		fmt.Fprintln(os.Stderr, "error: expected 'delete' as second argument")
		os.Exit(1)
	}

	db, err := localDb.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	inst, err := db.GetInstanceByAwsInstanceIDAndUserID(id, service.LocalUserID)
	if err == sql.ErrNoRows {
		fmt.Fprintf(os.Stderr, "error: box not found: %s\n", id)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if !inst.IdleStopMinutes.Valid {
		fmt.Fprintln(os.Stderr, "error: idle-stop is not set")
		os.Exit(1)
	}

	sshStatus, err := service.GetSshStatus(id, service.LocalUserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if !sshStatus.Ready {
		fmt.Fprintln(os.Stderr, "error: box is not ready yet (EC2 status checks still pending)")
		os.Exit(1)
	}
	if sshStatus.Instance == nil {
		fmt.Fprintln(os.Stderr, "error: box is ready but instance details are unavailable, try again in a few minutes")
		os.Exit(1)
	}
	box := sshStatus.Instance
	if box.Status != "running" {
		fmt.Fprintf(os.Stderr, "error: box is %s, not running\n", box.Status)
		os.Exit(1)
	}
	host := box.IPAddress
	if host == "" {
		host = box.PrivateIPAddress
	}
	if host == "" {
		fmt.Fprintln(os.Stderr, "error: box has no IP address (is it running?)")
		os.Exit(1)
	}

	sshBin, err := exec.LookPath("ssh")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: ssh binary not found in PATH")
		os.Exit(1)
	}

	identity := defaultKeyPath()
	ready, err := checkDevboxReady(sshBin, identity, "ec2-user", host, "22")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: readiness probe failed: %v\n", err)
		os.Exit(1)
	}
	if !ready {
		fmt.Fprintln(os.Stderr, "error: devbox is not ready yet — try again in a minute")
		os.Exit(1)
	}

	if err := uninstallIdleStop(sshBin, identity, "ec2-user", host); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := db.SetInstanceIdleStopMinutes(inst.AwsInstanceID, nil); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("idle-stop removed for %s\n", inst.Name)
}

func uninstallIdleStop(sshBin, identity, user, host string) error {
	script := `set -euo pipefail
systemctl disable --now devbox-idle-stop.timer 2>/dev/null || true
systemctl disable --now devbox-idle-stop-boot.service 2>/dev/null || true
systemctl daemon-reload
rm -f /etc/systemd/system/devbox-idle-stop.timer
rm -f /etc/systemd/system/devbox-idle-stop.service
rm -f /etc/systemd/system/devbox-idle-stop-boot.service
rm -f /usr/local/bin/devbox-idle-stop
rm -f /var/lib/devbox/idle-stop-minutes
rm -f /var/lib/devbox/last-activity
systemctl daemon-reload
systemctl reset-failed devbox-idle-stop.service 2>/dev/null || true
systemctl reset-failed devbox-idle-stop-boot.service 2>/dev/null || true
`

	target := fmt.Sprintf("%s@%s", user, host)
	argv := append([]string{sshBin}, sshBaseArgs(identity, "22")...)
	argv = append(argv, target, "sudo", "bash", "-s")

	cmd := execCommand(sshBin, argv[1:]...)
	cmd.Stdin = strings.NewReader(script)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("idle-stop uninstall failed: %s", msg)
	}
	return nil
}

// update for a specific instance
func updateIdleStop(args []string) {
	if TestMode {
		fmt.Println("[test] idle-stop update: done")
		return
	}

	if len(args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: devbox idle-stop <id> update <minutes>")
		os.Exit(1)
	}

	id := args[0]
	if id == "" {
		fmt.Fprintln(os.Stderr, "error: id is required")
		os.Exit(1)
	}
	if args[1] != "update" {
		fmt.Fprintln(os.Stderr, "error: expected 'update' as second argument")
		os.Exit(1)
	}

	minutesInt, err := strconv.Atoi(args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: minutes must be an integer")
		os.Exit(1)
	}
	if minutesInt <= 0 {
		fmt.Fprintln(os.Stderr, "error: minutes must be greater than 0")
		os.Exit(1)
	}

	db, err := localDb.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	inst, err := db.GetInstanceByAwsInstanceIDAndUserID(id, service.LocalUserID)
	if err == sql.ErrNoRows {
		fmt.Fprintf(os.Stderr, "error: box not found: %s\n", id)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if !inst.IdleStopMinutes.Valid {
		fmt.Fprintln(os.Stderr, "error: idle-stop is not set — use 'devbox idle-stop <id> in <minutes>' first")
		os.Exit(1)
	}

	sshStatus, err := service.GetSshStatus(id, service.LocalUserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if !sshStatus.Ready {
		fmt.Fprintln(os.Stderr, "error: box is not ready yet (EC2 status checks still pending)")
		os.Exit(1)
	}
	if sshStatus.Instance == nil {
		fmt.Fprintln(os.Stderr, "error: box is ready but instance details are unavailable, try again in a few minutes")
		os.Exit(1)
	}
	box := sshStatus.Instance
	if box.Status != "running" {
		fmt.Fprintf(os.Stderr, "error: box is %s, not running\n", box.Status)
		os.Exit(1)
	}
	host := box.IPAddress
	if host == "" {
		host = box.PrivateIPAddress
	}
	if host == "" {
		fmt.Fprintln(os.Stderr, "error: box has no IP address (is it running?)")
		os.Exit(1)
	}

	sshBin, err := exec.LookPath("ssh")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: ssh binary not found in PATH")
		os.Exit(1)
	}

	identity := defaultKeyPath()
	ready, err := checkDevboxReady(sshBin, identity, "ec2-user", host, "22")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: readiness probe failed: %v\n", err)
		os.Exit(1)
	}
	if !ready {
		fmt.Fprintln(os.Stderr, "error: devbox is not ready yet — try again in a minute")
		os.Exit(1)
	}

	if err := updateIdleStopOnHost(sshBin, identity, "ec2-user", host, minutesInt); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := db.SetInstanceIdleStopMinutes(inst.AwsInstanceID, &minutesInt); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("last-activity reset on host")
	fmt.Printf("idle-stop updated to %d minutes for %s\n", minutesInt, inst.Name)
}

func updateIdleStopOnHost(sshBin, identity, user, host string, minutes int) error {
	script := fmt.Sprintf(`set -euo pipefail
echo %d > /var/lib/devbox/idle-stop-minutes
date +%%s > /var/lib/devbox/last-activity
`, minutes)

	target := fmt.Sprintf("%s@%s", user, host)
	argv := append([]string{sshBin}, sshBaseArgs(identity, "22")...)
	argv = append(argv, target, "sudo", "bash", "-s")

	cmd := execCommand(sshBin, argv[1:]...)
	cmd.Stdin = strings.NewReader(script)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("idle-stop update failed: %s", msg)
	}
	return nil
}

// show for a specific instance
func showIdleStop(args []string) {
	if TestMode {
		fmt.Println("[test] idle-stop show: done")
		return
	}

	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: devbox idle-stop <id> show")
		os.Exit(1)
	}

	id := args[0]
	if args[1] != "show" {
		fmt.Fprintln(os.Stderr, "error: expected 'show' as second argument")
		os.Exit(1)
	}
	if err := validateEc2InstanceID(id); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	db, err := localDb.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	inst, err := db.GetInstanceByAwsInstanceIDAndUserID(id, service.LocalUserID)
	if err == sql.ErrNoRows {
		fmt.Fprintf(os.Stderr, "error: box not found: %s\n", id)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if !inst.IdleStopMinutes.Valid {
		fmt.Println("no idle stop set")
		return
	}
	fmt.Printf("idle-stop is set to %d minutes for %s\n", inst.IdleStopMinutes.Int64, inst.Name)
}
