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
	return selectVolumeSizeGBWithDefault(service.DefaultVolumeSizeGB, service.MinVolumeSizeGB)
}

func selectVolumeSizeGBWithDefault(defaultSizeGB, minSizeGB int) (int, error) {
	if !isTerminal(os.Stdin) {
		return defaultSizeGB, nil
	}

	fmt.Printf("Volume size in GB [%d]: ", defaultSizeGB)

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return defaultSizeGB, nil
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return defaultSizeGB, nil
	}

	size, err := strconv.Atoi(line)
	if err != nil {
		return 0, fmt.Errorf("invalid volume size %q: must be an integer", line)
	}
	if err := service.ValidateVolumeSizeGB(size); err != nil {
		return 0, err
	}
	if size < minSizeGB {
		return 0, fmt.Errorf("invalid volume size %d GB (must be at least current size %d GB)", size, minSizeGB)
	}
	return size, nil
}
