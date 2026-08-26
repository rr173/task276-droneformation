package state

import (
	"testing"

	"task276-droneformation/internal/model"
)

func TestSealedBlocksVerify(t *testing.T) {
	sealed := model.FormationRun{Status: model.RunSealed}
	if !IsSealed(sealed) || CanVerify(sealed) {
		t.Fatal("sealed run must not be verifiable")
	}
	open := model.FormationRun{Status: model.RunReceiving}
	if IsSealed(open) || !CanVerify(open) {
		t.Fatal("receiving run must be verifiable")
	}
}

func TestNextVerificationStatus(t *testing.T) {
	if NextVerificationStatus(true) != model.RunConflict {
		t.Fatal("conflict expected")
	}
	if NextVerificationStatus(false) != model.RunSafe {
		t.Fatal("safe expected")
	}
}
