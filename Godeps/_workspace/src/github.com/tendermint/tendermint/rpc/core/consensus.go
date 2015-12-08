package core

import (
	"github.com/eris-ltd/eris-db/Godeps/_workspace/src/github.com/tendermint/go-wire"
	cm "github.com/eris-ltd/eris-db/Godeps/_workspace/src/github.com/tendermint/tendermint/consensus"
	ctypes "github.com/eris-ltd/eris-db/Godeps/_workspace/src/github.com/tendermint/tendermint/rpc/core/types"
	"github.com/eris-ltd/eris-db/Godeps/_workspace/src/github.com/tendermint/tendermint/types"
)

func ListValidators() (*ctypes.ResultListValidators, error) {
	var blockHeight int
	var validators []*types.Validator

	state := consensusState.GetState()
	blockHeight = state.LastBlockHeight
	state.Validators.Iterate(func(index int, val *types.Validator) bool {
		validators = append(validators, val)
		return false
	})

	return &ctypes.ResultListValidators{blockHeight, validators}, nil
}

func DumpConsensusState() (*ctypes.ResultDumpConsensusState, error) {
	roundState := consensusState.GetRoundState()
	peerRoundStates := []string{}
	for _, peer := range p2pSwitch.Peers().List() {
		// TODO: clean this up?
		peerState := peer.Data.Get(types.PeerStateKey).(*cm.PeerState)
		peerRoundState := peerState.GetRoundState()
		peerRoundStateStr := peer.Key + ":" + string(wire.JSONBytes(peerRoundState))
		peerRoundStates = append(peerRoundStates, peerRoundStateStr)
	}
	return &ctypes.ResultDumpConsensusState{roundState.String(), peerRoundStates}, nil
}
