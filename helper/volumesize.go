package helper

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"outpost-cli/service"
)

// helper function to select the volume size in GB

// selectVolumeSizeGB selects the volume size in GB
func SelectVolumeSizeGB() (int, error) {
	return SelectVolumeSizeGBWithDefault(service.DefaultVolumeSizeGB, service.MinVolumeSizeGB)
}

// inputs are the default size and the minimum size
// output is the selected size and error
// to make sure the size is within the allowed range
func SelectVolumeSizeGBWithDefault(defaultSizeGB, minSizeGB int) (int, error) {
	if !IsTerminal(os.Stdin) {
		return defaultSizeGB, nil
	}

	fmt.Printf("Volume size in GB (min %d, max %d; Enter to use %d): ", minSizeGB, service.MaxVolumeSizeGB, defaultSizeGB)

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
