package analysis

import (
	"os"
	"testing"

	"token-volume-tracker/pkg/utils"
)

// TestMain ensures all tests run in the project root directory
func TestMain(m *testing.M) {
	// Get the project root directory
	root, err := utils.GetProjectRoot()
	if err != nil {
		panic("Failed to get project root: " + err.Error())
	}

	// Change to project root directory
	if err := os.Chdir(root); err != nil {
		panic("Failed to change to project root: " + err.Error())
	}

	// Run the tests
	os.Exit(m.Run())
}
