//go:build test

package config

// Test data constants - only available during testing
const (
	// Mock user IDs
	TestUserID = 123

	// Mock performance data
	MockCalculationsPerSecond = 10.5
	MockAvgCalculationTime    = 0.05
	MockAvgQueryTime          = 0.02
	MockMemoryUsage           = 128.5
	MockAvgScore              = 150.0
	MockPriorityUpdates       = 100
	MockQueueSize             = 5

	// Mock question counts
	MockTotalAttempts    = 50
	MockCorrectAttempts  = 30
	MockTotalAttempts2   = 40
	MockCorrectAttempts2 = 25

	// Mock availability and demand
	MockAvailable1 = 100
	MockDemand1    = 20
	MockAvailable2 = 80
	MockDemand2    = 15
	MockCount      = 50
)
