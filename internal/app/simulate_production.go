//go:build production

package app

// Release builds carry no update simulation.
func updateSimulationActive() bool { return false }

func simulateUpdates(*updateCapsule) {}
