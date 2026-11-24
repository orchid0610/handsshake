package vault

import (
	"errors"
	"fmt"
)

// MaxJumpHops is how many jump hosts may be chained in a row.
const MaxJumpHops = 4

var (
	ErrJumpMissing = errors.New("the jump host no longer exists")
	ErrJumpCycle   = errors.New("a host can't connect through itself: these jump hosts form a loop")
	ErrJumpTooDeep = fmt.Errorf("too many jump hosts in a row (at most %d)", MaxJumpHops)
)

// JumpChain returns the hosts to pass through, outermost jump host first and
// ending with h itself. hosts is where jump hosts are looked up by ID. h may be
// a host that isn't saved yet.
func JumpChain(hosts []Host, h Host) ([]Host, error) {
	byID := make(map[string]Host, len(hosts))
	for _, x := range hosts {
		byID[x.ID] = x
	}

	chain := []Host{h}
	seen := map[string]bool{h.ID: true}
	cur := h
	for cur.JumpHostID != "" {
		if seen[cur.JumpHostID] {
			return nil, ErrJumpCycle
		}
		next, ok := byID[cur.JumpHostID]
		if !ok {
			return nil, ErrJumpMissing
		}
		seen[next.ID] = true
		chain = append([]Host{next}, chain...)
		if len(chain)-1 > MaxJumpHops {
			return nil, ErrJumpTooDeep
		}
		cur = next
	}
	return chain, nil
}