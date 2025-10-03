package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion_Variables(t *testing.T) {
	// Test that version variables are defined and are strings
	assert.NotEmpty(t, Version)
	assert.IsType(t, "", Version)

	assert.NotEmpty(t, Commit)
	assert.IsType(t, "", Commit)

	assert.NotEmpty(t, BuildTime)
	assert.IsType(t, "", BuildTime)
}

func TestVersion_DefaultValues(t *testing.T) {
	// Test default values for development
	// These should be the default values set at build time
	assert.Equal(t, "dev", Version)
	assert.Equal(t, "dev", Commit)
	assert.Equal(t, "unknown", BuildTime)
}

func TestVersion_VariableTypes(t *testing.T) {
	// Test that variables are of correct types
	var version string = Version
	var commit string = Commit
	var buildTime string = BuildTime

	// These should compile and assign correctly
	assert.IsType(t, "", version)
	assert.IsType(t, "", commit)
	assert.IsType(t, "", buildTime)
}

func TestVersion_VariableAccessibility(t *testing.T) {
	// Test that version variables are accessible and can be used
	versionInfo := map[string]string{
		"Version":   Version,
		"Commit":    Commit,
		"BuildTime": BuildTime,
	}

	assert.Len(t, versionInfo, 3)
	assert.Contains(t, versionInfo, "Version")
	assert.Contains(t, versionInfo, "Commit")
	assert.Contains(t, versionInfo, "BuildTime")

	// All values should be non-empty strings
	for key, value := range versionInfo {
		assert.NotEmpty(t, value, "Version variable %s should not be empty", key)
		assert.IsType(t, "", value, "Version variable %s should be a string", key)
	}
}

func TestVersion_UsageInStruct(t *testing.T) {
	// Test that version variables can be used in structs
	type AppInfo struct {
		Version   string
		Commit    string
		BuildTime string
	}

	appInfo := AppInfo{
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
	}

	assert.Equal(t, Version, appInfo.Version)
	assert.Equal(t, Commit, appInfo.Commit)
	assert.Equal(t, BuildTime, appInfo.BuildTime)
}

func TestVersion_UsageInJSON(t *testing.T) {
	// Test that version variables can be used in JSON responses
	type VersionResponse struct {
		Version   string `json:"version"`
		Commit    string `json:"commit"`
		BuildTime string `json:"build_time"`
	}

	response := VersionResponse{
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
	}

	assert.Equal(t, Version, response.Version)
	assert.Equal(t, Commit, response.Commit)
	assert.Equal(t, BuildTime, response.BuildTime)
}
