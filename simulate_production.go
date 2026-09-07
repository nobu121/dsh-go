//go:build production

package main

// Release builds carry no update simulation.
func updateSimulationActive() bool { return false }

func simulateUpdates(*updateCapsule) {}
