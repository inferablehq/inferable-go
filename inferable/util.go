package inferable

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

const MachineIDFile = "inferable_machine_id.json"

type machineIDData struct {
	ID string `json:"id"`
}

func generateMachineID() (string, error) {
	tmpDir := os.TempDir()
	machineIDPath := filepath.Join(tmpDir, MachineIDFile)

	// Try to read existing machine ID
	data, err := os.ReadFile(machineIDPath)
	if err == nil {
		var storedID machineIDData
		if err := json.Unmarshal(data, &storedID); err == nil && storedID.ID != "" {
			return storedID.ID, nil
		}
	}

	// Generate new machine ID if not found or invalid
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %v", err)
	}

	uniqueID, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("failed to generate UUID: %v", err)
	}

	machineID := fmt.Sprintf("%s-%s", hostname, uniqueID.String())

	// Store the new machine ID
	newData := machineIDData{ID: machineID}
	jsonData, err := json.Marshal(newData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal machine ID: %v", err)
	}

	if err := os.WriteFile(machineIDPath, jsonData, 0600); err != nil {
		return "", fmt.Errorf("failed to write machine ID file: %v", err)
	}

	return machineID, nil
}
