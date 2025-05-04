package cap

import (
	"testing"
	"time"
)

func TestNewCAPOrchestrator(t *testing.T) {
	co := NewCAPOrchestrator()

	if co == nil {
		t.Fatal("NewCAPOrchestrator returned nil")
	}

	if co.currentProfile.ConsistencyLevel != StrongConsistency {
		t.Error("Default consistency level should be StrongConsistency")
	}
}

func TestConsistencyLevelAdjustment(t *testing.T) {
	co := NewCAPOrchestrator()

	// Test optimal consistency calculation
	level := co.calculateOptimalConsistency(0.8) // High partition risk
	if level != EventualConsistency {
		t.Errorf("Expected EventualConsistency for high risk, got %v", level)
	}

	level = co.calculateOptimalConsistency(0.1) // Low partition risk
	if level != StrongConsistency {
		t.Errorf("Expected StrongConsistency for low risk, got %v", level)
	}
}

func TestConflictResolver(t *testing.T) {
	cr := NewConflictResolver()

	// Create test conflict
	conflict := &StateConflict{
		States: []ConflictingState{
			{
				NodeID:    "node1",
				State:     "state1",
				Timestamp: time.Now(),
				VectorClock: &VectorClock{
					Clocks: map[string]uint64{"node1": 1},
				},
			},
			{
				NodeID:    "node2",
				State:     "state2",
				Timestamp: time.Now().Add(time.Second),
				VectorClock: &VectorClock{
					Clocks: map[string]uint64{"node2": 1},
				},
			},
		},
	}

	// Detect conflict type
	conflictType := cr.DetectConflictType(conflict)
	if conflictType != ConcurrentWrite {
		t.Errorf("Expected ConcurrentWrite, got %v", conflictType)
	}

	// Resolve conflict
	resolved, err := cr.ResolveConcurrentWrites(conflict)
	if err != nil {
		t.Errorf("Failed to resolve conflict: %v", err)
	}

	if resolved == nil {
		t.Error("Resolved state is nil")
	}
}

func TestVectorClock(t *testing.T) {
	cr := NewConflictResolver()

	// Test happens-before relationship
	vc1 := &VectorClock{
		Clocks: map[string]uint64{"node1": 1, "node2": 2},
	}

	vc2 := &VectorClock{
		Clocks: map[string]uint64{"node1": 2, "node2": 3},
	}

	if !cr.happensBefore(vc1, vc2) {
		t.Error("vc1 should happen before vc2")
	}

	if cr.happensBefore(vc2, vc1) {
		t.Error("vc2 should not happen before vc1")
	}
}

func TestPartitionDetection(t *testing.T) {
	pd := NewPartitionDetector()

	// Test partition risk prediction
	risk := pd.PredictPartitionRisk()
	if risk < 0 || risk > 1 {
		t.Errorf("Partition risk should be between 0 and 1, got %v", risk)
	}

	// Test partition detection
	partitions := pd.DetectPartitions()
	if partitions == nil {
		t.Error("DetectPartitions returned nil")
	}
}
