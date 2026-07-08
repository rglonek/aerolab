package jobqueue

import (
	"errors"
	"testing"
)

func TestSimpleQueueAddFull(t *testing.T) {
	// concurrent=1, queued=2 => maxJobs=3
	q := NewSimpleQueue(1, 2)
	for i := 0; i < 3; i++ {
		if err := q.Add(); err != nil {
			t.Fatalf("Add #%d returned %v, want nil", i, err)
		}
	}
	if err := q.Add(); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("Add over capacity = %v, want ErrQueueFull", err)
	}
}

func TestSimpleQueueSetNoAccept(t *testing.T) {
	q := NewSimpleQueue(2, 2)
	sentinel := errors.New("shutting down")
	q.SetNoAccept(sentinel)
	if err := q.Add(); !errors.Is(err, sentinel) {
		t.Fatalf("Add after SetNoAccept = %v, want sentinel", err)
	}
}

func TestSimpleQueueRemoveEmpty(t *testing.T) {
	q := NewSimpleQueue(1, 1)
	if err := q.Remove(); !errors.Is(err, ErrQueueEmpty) {
		t.Fatalf("Remove on empty = %v, want ErrQueueEmpty", err)
	}
}

func TestSimpleQueueEndEmpty(t *testing.T) {
	q := NewSimpleQueue(1, 1)
	if err := q.End(); !errors.Is(err, ErrQueueEmpty) {
		t.Fatalf("End with no running jobs = %v, want ErrQueueEmpty", err)
	}
}

func TestSimpleQueueSizeTracking(t *testing.T) {
	q := NewSimpleQueue(2, 2)
	if err := q.Add(); err != nil {
		t.Fatalf("Add: %v", err)
	}
	concurrent, queued := q.GetSize()
	if concurrent != 0 || queued != 1 {
		t.Fatalf("after Add GetSize = (%d,%d), want (0,1)", concurrent, queued)
	}
	q.Start()
	concurrent, queued = q.GetSize()
	if concurrent != 1 || queued != 0 {
		t.Fatalf("after Start GetSize = (%d,%d), want (1,0)", concurrent, queued)
	}
	if err := q.End(); err != nil {
		t.Fatalf("End: %v", err)
	}
	if err := q.Remove(); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	concurrent, queued = q.GetSize()
	if concurrent != 0 || queued != 0 {
		t.Fatalf("after End+Remove GetSize = (%d,%d), want (0,0)", concurrent, queued)
	}
}

func TestQueueWithIDsLifecycle(t *testing.T) {
	q := NewQueueWithIDs(1, 1)

	if err := q.Add("job1"); err != nil {
		t.Fatalf("Add job1: %v", err)
	}
	if status, ok := q.GetJobStatus("job1"); !ok || status != JobCreated {
		t.Fatalf("job1 status = (%v,%v), want (JobCreated,true)", status, ok)
	}
	if err := q.Add("job1"); !errors.Is(err, ErrJobAlreadyQueued) {
		t.Fatalf("re-Add job1 = %v, want ErrJobAlreadyQueued", err)
	}

	if err := q.Start("job1"); err != nil {
		t.Fatalf("Start job1: %v", err)
	}
	if status, _ := q.GetJobStatus("job1"); status != JobRunning {
		t.Fatalf("job1 status after Start = %v, want JobRunning", status)
	}
	if err := q.Start("job1"); !errors.Is(err, ErrJobAlreadyStarted) {
		t.Fatalf("re-Start job1 = %v, want ErrJobAlreadyStarted", err)
	}

	if err := q.Remove("job1"); !errors.Is(err, ErrJobRunning) {
		t.Fatalf("Remove running job = %v, want ErrJobRunning", err)
	}

	if err := q.End("job1"); err != nil {
		t.Fatalf("End job1: %v", err)
	}
	if status, _ := q.GetJobStatus("job1"); status != JobFinished {
		t.Fatalf("job1 status after End = %v, want JobFinished", status)
	}

	if err := q.Remove("job1"); err != nil {
		t.Fatalf("Remove finished job: %v", err)
	}
	if _, ok := q.GetJobStatus("job1"); ok {
		t.Fatal("job1 should be gone after Remove")
	}
}

func TestQueueWithIDsErrors(t *testing.T) {
	q := NewQueueWithIDs(1, 1)
	if err := q.Start("ghost"); !errors.Is(err, ErrJobNotFound) {
		t.Fatalf("Start unknown = %v, want ErrJobNotFound", err)
	}
	if err := q.End("ghost"); !errors.Is(err, ErrJobNotFound) {
		t.Fatalf("End unknown = %v, want ErrJobNotFound", err)
	}
	if err := q.Remove("ghost"); !errors.Is(err, ErrJobNotFound) {
		t.Fatalf("Remove unknown = %v, want ErrJobNotFound", err)
	}
	if err := q.Add("j"); err != nil {
		t.Fatalf("Add j: %v", err)
	}
	if err := q.End("j"); !errors.Is(err, ErrJobNotRunning) {
		t.Fatalf("End queued-not-running = %v, want ErrJobNotRunning", err)
	}
}

func TestQueueWithIDsGetJobsIsCopy(t *testing.T) {
	q := NewQueueWithIDs(2, 2)
	if err := q.Add("a"); err != nil {
		t.Fatalf("Add a: %v", err)
	}
	jobs := q.GetJobs()
	jobs["a"] = JobFinished
	jobs["injected"] = JobRunning
	if status, _ := q.GetJobStatus("a"); status != JobCreated {
		t.Fatalf("mutating GetJobs affected queue: job a = %v", status)
	}
	if _, ok := q.GetJobStatus("injected"); ok {
		t.Fatal("mutating GetJobs injected a phantom job into the queue")
	}
}
