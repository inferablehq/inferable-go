package inferable

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"strings"
)

const MachineIDFile = "inferable_machine_id.json"

type machineIDData struct {
	ID string `json:"id"`
}

func getMachineID() string {
	hostname, _ := os.Hostname()
	cpuInfo := runtime.GOARCH + runtime.GOOS + runtime.Version()
	machineID := hostname + cpuInfo

	hash := sha256.Sum256([]byte(machineID))
	return hex.EncodeToString(hash[:])
}

func generateMachineID(length int) string {
	machineID := getMachineID()
	seed := int64(0)
	for _, char := range machineID {
		seed += int64(char)
	}

	r := rand.New(rand.NewSource(seed))
	const charset = "abcdefghijklmnopqrstuvwxyz"

	var sb strings.Builder
	sb.Grow(length)
	for i := 0; i < length; i++ {
		sb.WriteByte(charset[r.Intn(len(charset))])
	}

	return fmt.Sprintf("go-%s", sb.String())
}
