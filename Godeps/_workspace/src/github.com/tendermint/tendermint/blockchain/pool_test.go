package blockchain

import (
	"math/rand"
	"testing"
	"time"

	. "github.com/eris-ltd/eris-db/Godeps/_workspace/src/github.com/tendermint/tendermint/common"
	"github.com/eris-ltd/eris-db/Godeps/_workspace/src/github.com/tendermint/tendermint/types"
)

type testPeer struct {
	id     string
	height uint
}

func makePeers(numPeers int, minHeight, maxHeight uint) map[string]testPeer {
	peers := make(map[string]testPeer, numPeers)
	for i := 0; i < numPeers; i++ {
		peerId := RandStr(12)
		height := minHeight + uint(rand.Intn(int(maxHeight-minHeight)))
		peers[peerId] = testPeer{peerId, height}
	}
	return peers
}

func TestBasic(t *testing.T) {
	peers := makePeers(10, 0, 1000)
	start := uint(42)
	timeoutsCh := make(chan string, 100)
	requestsCh := make(chan BlockRequest, 100)
	pool := NewBlockPool(start, requestsCh, timeoutsCh)
	pool.Start()

	// Introduce each peer.
	go func() {
		for _, peer := range peers {
			pool.SetPeerHeight(peer.id, peer.height)
		}
	}()

	// Start a goroutine to pull blocks
	go func() {
		for {
			if !pool.IsRunning() {
				return
			}
			first, second := pool.PeekTwoBlocks()
			if first != nil && second != nil {
				pool.PopRequest()
			} else {
				time.Sleep(1 * time.Second)
			}
		}
	}()

	// Pull from channels
	for {
		select {
		case peerId := <-timeoutsCh:
			t.Errorf("timeout: %v", peerId)
		case request := <-requestsCh:
			log.Debug("TEST: Pulled new BlockRequest", "request", request)
			if request.Height == 300 {
				return // Done!
			}
			// Request desired, pretend like we got the block immediately.
			go func() {
				block := &types.Block{Header: &types.Header{Height: request.Height}}
				pool.AddBlock(block, request.PeerId)
				log.Debug("TEST: Added block", "block", request.Height, "peer", request.PeerId)
			}()
		}
	}

	pool.Stop()
}

func TestTimeout(t *testing.T) {
	peers := makePeers(10, 0, 1000)
	start := uint(42)
	timeoutsCh := make(chan string, 100)
	requestsCh := make(chan BlockRequest, 100)
	pool := NewBlockPool(start, requestsCh, timeoutsCh)
	pool.Start()

	// Introduce each peer.
	go func() {
		for _, peer := range peers {
			pool.SetPeerHeight(peer.id, peer.height)
		}
	}()

	// Start a goroutine to pull blocks
	go func() {
		for {
			if !pool.IsRunning() {
				return
			}
			first, second := pool.PeekTwoBlocks()
			if first != nil && second != nil {
				pool.PopRequest()
			} else {
				time.Sleep(1 * time.Second)
			}
		}
	}()

	// Pull from channels
	counter := 0
	timedOut := map[string]struct{}{}
	for {
		select {
		case peerId := <-timeoutsCh:
			log.Debug("Timeout", "peerId", peerId)
			if _, ok := timedOut[peerId]; !ok {
				counter++
				if counter == len(peers) {
					return // Done!
				}
			}
		case request := <-requestsCh:
			log.Debug("TEST: Pulled new BlockRequest", "request", request)
		}
	}

	pool.Stop()
}
