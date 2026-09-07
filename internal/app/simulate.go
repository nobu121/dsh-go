//go:build !production

package app

// Dev-only update simulation. The real updater is disabled without an
// UpdateRepo, so these env hooks are the only way to exercise the prep-page
// offer in `wails3 task build DEV=true`:
//
//	DSH_SIMULATE_UPDATE=9.9.9          client offer on the load screen
//	DSH_SIMULATE_DSH_UPDATE=9.9.9      runtime offer on the load screen
//
// Applying an offer downloads from the faster of CNB and GitHub. A simulated
// version that does not exist falls back to the latest real release.

import (
	"log"
	"os"
	"strings"
)

func updateSimulationActive() bool {
	return strings.TrimSpace(os.Getenv("DSH_SIMULATE_UPDATE")) != "" ||
		strings.TrimSpace(os.Getenv("DSH_SIMULATE_DSH_UPDATE")) != ""
}

func simulateUpdates(offer *updateCapsule) {
	if offer == nil {
		return
	}
	dshVer := strings.TrimSpace(os.Getenv("DSH_SIMULATE_DSH_UPDATE"))
	shellVer := strings.TrimSpace(os.Getenv("DSH_SIMULATE_UPDATE"))
	if dshVer == "" && shellVer == "" {
		return
	}
	if shellVer != "" {
		log.Printf("simulating shell update: %s", shellVer)
		offer.stashApp(shellVer)
	}
	if dshVer != "" {
		log.Printf("simulating dsh update: %s", dshVer)
		offer.showDSH(dshVer)
	}
}
