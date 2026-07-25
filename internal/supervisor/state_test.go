package supervisor

import (
	"fmt"
	"sync"
	"testing"
)

func TestStateTrackerRegisterAndGet(t *testing.T) {
	st := NewStateTracker()
	st.Register("worker-a")

	info, ok := st.Get("worker-a")
	if !ok {
		t.Fatal("Get() = false para proceso registrado")
	}
	if info.Name != "worker-a" {
		t.Fatalf("Name = %q, want worker-a", info.Name)
	}
	if info.State != StateIdle {
		t.Fatalf("State = %v, want StateIdle", info.State)
	}
}

func TestStateTrackerGetAll(t *testing.T) {
	st := NewStateTracker()
	st.Register("a")
	st.Register("b")
	st.Register("c")

	all := st.GetAll()
	if len(all) != 3 {
		t.Fatalf("GetAll() returned %d procesos, want 3", len(all))
	}
}

func TestStateTrackerSetState(t *testing.T) {
	st := NewStateTracker()
	st.Register("worker")
	st.SetState("worker", StateRunning)

	info, _ := st.Get("worker")
	if info.State != StateRunning {
		t.Fatalf("State = %v, want StateRunning", info.State)
	}
	if info.StartedAt.IsZero() {
		t.Fatal("StartedAt debería haberse actualizado al pasar a Running")
	}
}

func TestStateTrackerSetStateStopped(t *testing.T) {
	st := NewStateTracker()
	st.Register("worker")
	st.SetState("worker", StateStopped)

	info, _ := st.Get("worker")
	if info.StoppedAt.IsZero() {
		t.Fatal("StoppedAt debería haberse actualizado al pasar a Stopped")
	}
}

func TestStateTrackerSetError(t *testing.T) {
	st := NewStateTracker()
	st.Register("worker")

	err := fmt.Errorf("proceso terminó con código 1")
	st.SetError("worker", err, 1)

	info, _ := st.Get("worker")
	if info.State != StateFailed {
		t.Fatalf("State = %v, want StateFailed", info.State)
	}
	if info.LastExitCode != 1 {
		t.Fatalf("LastExitCode = %d, want 1", info.LastExitCode)
	}
	if info.Error == nil {
		t.Fatal("Error debería no ser nil")
	}
}

func TestStateTrackerIncrementAndResetRetries(t *testing.T) {
	st := NewStateTracker()
	st.Register("worker")

	st.IncrementRetries("worker")
	st.IncrementRetries("worker")
	st.IncrementRetries("worker")

	info, _ := st.Get("worker")
	if info.Retries != 3 {
		t.Fatalf("Retries = %d, want 3", info.Retries)
	}

	st.ResetRetries("worker")
	info, _ = st.Get("worker")
	if info.Retries != 0 {
		t.Fatalf("Retries después de Reset = %d, want 0", info.Retries)
	}
}

func TestStateTrackerSummary(t *testing.T) {
	st := NewStateTracker()
	st.Register("a")
	st.Register("b")
	st.SetState("a", StateRunning)
	st.SetState("b", StateFailed)

	summary := st.Summary()
	if summary["a"] != "running" {
		t.Fatalf("Summary[a] = %q, want running", summary["a"])
	}
	if summary["b"] != "failed" {
		t.Fatalf("Summary[b] = %q, want failed", summary["b"])
	}
}

func TestStateTrackerRunningAndFailedCount(t *testing.T) {
	st := NewStateTracker()
	st.Register("a")
	st.Register("b")
	st.Register("c")
	st.Register("d")

	st.SetState("a", StateRunning)
	st.SetState("b", StateRunning)
	st.SetState("c", StateFailed)
	// d queda en StateIdle

	if got := st.RunningCount(); got != 2 {
		t.Fatalf("RunningCount() = %d, want 2", got)
	}
	if got := st.FailedCount(); got != 1 {
		t.Fatalf("FailedCount() = %d, want 1", got)
	}
}

func TestStateTrackerConcurrentAccess(t *testing.T) {
	st := NewStateTracker()
	const goroutines = 50

	for i := 0; i < goroutines; i++ {
		st.Register(fmt.Sprintf("w%d", i))
	}

	var wg sync.WaitGroup
	wg.Add(goroutines * 4)

	for i := 0; i < goroutines; i++ {
		name := fmt.Sprintf("w%d", i)

		go func() {
			defer wg.Done()
			st.SetState(name, StateRunning)
		}()
		go func() {
			defer wg.Done()
			st.IncrementRetries(name)
		}()
		go func() {
			defer wg.Done()
			st.Get(name)
		}()
		go func() {
			defer wg.Done()
			st.Summary()
		}()
	}

	wg.Wait()

	_ = st.RunningCount()
	_ = st.FailedCount()
}

