package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"devbox-cli/service"
)

func selectVolumeSizeGB() (int, error) {
	if !isTerminal(os.Stdin) {
		return service.DefaultVolumeSizeGB, nil
	}

	fmt.Printf("Volume size in GB [%d]: ", service.DefaultVolumeSizeGB)

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return service.DefaultVolumeSizeGB, nil
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return service.DefaultVolumeSizeGB, nil
	}

	size, err := strconv.Atoi(line)
	if err != nil {
		return 0, fmt.Errorf("invalid volume size %q: must be an integer", line)
	}
	if err := service.ValidateVolumeSizeGB(size); err != nil {
		return 0, err
	}
	return size, nil
}
