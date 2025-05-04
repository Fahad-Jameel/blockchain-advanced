package cap

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// ConflictResolver handles state conflict resolution
type ConflictResolver struct {
	vectorClocks    map[string]*VectorClock
	conflictLog     []*ConflictEntry
	resolutionRules map[ConflictType]ResolutionStrategy
}

// ConflictType represents the type of conflict
type ConflictType int

const (
	ConcurrentWrite ConflictType = iota
	PartitionDivergence
	CausalViolation
	UnknownConflict
)

// ConflictEntry represents a logged conflict
type ConflictEntry struct {
	Type         ConflictType
	Timestamp    time.Time
	Participants []string
	Resolution   string
}

// VectorClock represents a vector clock for causal ordering
type VectorClock struct {
	Clocks map[string]uint64
}

// StateConflict represents a state conflict
type StateConflict struct {
	States    []ConflictingState
	Type      ConflictType
	Timestamp time.Time
}

// ConflictingState represents a conflicting state version
type ConflictingState struct {
	NodeID      string
	State       interface{}
	VectorClock *VectorClock
	Timestamp   time.Time
}

// ResolvedState represents the resolved state
type ResolvedState struct {
	State       interface{}
	VectorClock *VectorClock
	Resolution  string
}

// ResolutionStrategy defines how to resolve conflicts
type ResolutionStrategy func(*StateConflict) (*ResolvedState, error)

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver() *ConflictResolver {
	cr := &ConflictResolver{
		vectorClocks:    make(map[string]*VectorClock),
		conflictLog:     make([]*ConflictEntry, 0),
		resolutionRules: make(map[ConflictType]ResolutionStrategy),
	}

	// Set up resolution strategies
	cr.resolutionRules[ConcurrentWrite] = cr.lastWriterWins
	cr.resolutionRules[PartitionDivergence] = cr.mergeStrategy
	cr.resolutionRules[CausalViolation] = cr.causalStrategy

	return cr
}

// DetectConflictType analyzes a conflict to determine its type
func (cr *ConflictResolver) DetectConflictType(conflict *StateConflict) ConflictType {
	// Fix: Properly detect concurrent writes
	if conflict == nil || len(conflict.States) == 0 {
		return UnknownConflict
	}

	// Check for concurrent writes (multiple states from different nodes)
	if len(conflict.States) > 1 {
		nodeMap := make(map[string]bool)
		for _, state := range conflict.States {
			nodeMap[state.NodeID] = true
		}

		// If we have multiple different nodes, it's a concurrent write
		if len(nodeMap) > 1 {
			return ConcurrentWrite
		}
	}

	// Use entropy analysis for partition divergence
	entropy := cr.calculateEntropy(conflict)
	if entropy > 0.8 {
		return PartitionDivergence
	}

	// Check for causal violations
	if cr.hasCausalViolation(conflict) {
		return CausalViolation
	}

	// Default case: concurrent write if multiple states
	if len(conflict.States) > 1 {
		return ConcurrentWrite
	}

	return UnknownConflict
}

// ResolveConcurrentWrites resolves concurrent write conflicts
func (cr *ConflictResolver) ResolveConcurrentWrites(conflict *StateConflict) (*ResolvedState, error) {
	return cr.resolutionRules[ConcurrentWrite](conflict)
}

// ResolvePartitionDivergence resolves partition divergence
func (cr *ConflictResolver) ResolvePartitionDivergence(conflict *StateConflict) (*ResolvedState, error) {
	return cr.resolutionRules[PartitionDivergence](conflict)
}

// ResolveCausalViolation resolves causal ordering violations
func (cr *ConflictResolver) ResolveCausalViolation(conflict *StateConflict) (*ResolvedState, error) {
	return cr.resolutionRules[CausalViolation](conflict)
}

// DefaultResolution provides a default resolution strategy
func (cr *ConflictResolver) DefaultResolution(conflict *StateConflict) (*ResolvedState, error) {
	return cr.lastWriterWins(conflict)
}

// ReconcileVectorClocks reconciles vector clocks after resolution
func (cr *ConflictResolver) ReconcileVectorClocks(resolved *ResolvedState) {
	// Merge all vector clocks
	merged := &VectorClock{
		Clocks: make(map[string]uint64),
	}

	for nodeID, clock := range resolved.VectorClock.Clocks {
		merged.Clocks[nodeID] = clock
	}

	resolved.VectorClock = merged
}

// Resolution strategies

func (cr *ConflictResolver) lastWriterWins(conflict *StateConflict) (*ResolvedState, error) {
	// Sort by timestamp
	sort.Slice(conflict.States, func(i, j int) bool {
		return conflict.States[i].Timestamp.After(conflict.States[j].Timestamp)
	})

	// Take the latest state
	winner := conflict.States[0]

	return &ResolvedState{
		State:       winner.State,
		VectorClock: winner.VectorClock,
		Resolution:  "last_writer_wins",
	}, nil
}

func (cr *ConflictResolver) mergeStrategy(conflict *StateConflict) (*ResolvedState, error) {
	// Implement a merge strategy for divergent states
	merged := &ResolvedState{
		State:       conflict.States[0].State, // Simplified merge
		VectorClock: cr.mergeVectorClocks(conflict),
		Resolution:  "merged",
	}

	return merged, nil
}

func (cr *ConflictResolver) causalStrategy(conflict *StateConflict) (*ResolvedState, error) {
	// Sort by causal order
	sorted := cr.sortByCausalOrder(conflict.States)

	// Take the causally latest state
	winner := sorted[len(sorted)-1]

	return &ResolvedState{
		State:       winner.State,
		VectorClock: winner.VectorClock,
		Resolution:  "causal_order",
	}, nil
}

// Helper methods

func (cr *ConflictResolver) calculateEntropy(conflict *StateConflict) float64 {
	// Calculate Shannon entropy of the conflicting states
	frequencies := make(map[string]int)
	total := 0

	for _, state := range conflict.States {
		// Use a hash or serialization of the state
		key := fmt.Sprintf("%v", state.State)
		frequencies[key]++
		total++
	}

	entropy := 0.0
	for _, freq := range frequencies {
		p := float64(freq) / float64(total)
		entropy -= p * math.Log2(p)
	}

	return entropy
}

func (cr *ConflictResolver) hasCausalViolation(conflict *StateConflict) bool {
	for i := 0; i < len(conflict.States); i++ {
		for j := i + 1; j < len(conflict.States); j++ {
			if !cr.areClocksCausallyConsistent(
				conflict.States[i].VectorClock,
				conflict.States[j].VectorClock,
			) {
				return true
			}
		}
	}
	return false
}

func (cr *ConflictResolver) areClocksCausallyConsistent(vc1, vc2 *VectorClock) bool {
	// Check if vector clocks are causally consistent
	concurrent := true

	for nodeID, clock1 := range vc1.Clocks {
		clock2, exists := vc2.Clocks[nodeID]
		if exists && clock1 > clock2 {
			concurrent = false
			break
		}
	}

	return concurrent
}

func (cr *ConflictResolver) mergeVectorClocks(conflict *StateConflict) *VectorClock {
	merged := &VectorClock{
		Clocks: make(map[string]uint64),
	}

	for _, state := range conflict.States {
		for nodeID, clock := range state.VectorClock.Clocks {
			if current, exists := merged.Clocks[nodeID]; !exists || clock > current {
				merged.Clocks[nodeID] = clock
			}
		}
	}

	return merged
}

func (cr *ConflictResolver) sortByCausalOrder(states []ConflictingState) []ConflictingState {
	// Implement topological sort based on vector clocks
	sort.Slice(states, func(i, j int) bool {
		return cr.happensBefore(states[i].VectorClock, states[j].VectorClock)
	})

	return states
}

func (cr *ConflictResolver) happensBefore(vc1, vc2 *VectorClock) bool {
	// Check if vc1 happens before vc2
	less := false
	equal := true

	for nodeID, clock1 := range vc1.Clocks {
		clock2, exists := vc2.Clocks[nodeID]
		if !exists || clock1 > clock2 {
			return false
		}
		if clock1 < clock2 {
			less = true
		}
		if clock1 != clock2 {
			equal = false
		}
	}

	return less && !equal
}
