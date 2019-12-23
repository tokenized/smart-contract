package scheduler

import (
	"context"
	"time"
)

type PeriodicProcessInterface interface {
	Run(context.Context)
}

// PeriodicProcess is a Scheduler job runs a process at a specified frequency.
type PeriodicProcess struct {
	name      string
	process   PeriodicProcessInterface
	frequency time.Duration
	next      time.Time
}

func NewPeriodicProcess(name string, process PeriodicProcessInterface, frequency time.Duration) *PeriodicProcess {
	return &PeriodicProcess{
		name:      name,
		process:   process,
		frequency: frequency,
		next:      time.Now().Add(frequency),
	}
}

// IsReady returns true when a job should be executed.
func (pp *PeriodicProcess) IsReady(ctx context.Context) bool {
	return time.Now().After(pp.next)
}

// Run executes the job.
func (pp *PeriodicProcess) Run(ctx context.Context) {
	// Schedule next time
	pp.next = time.Now().Add(pp.frequency)

	// Run process
	pp.process.Run(ctx)
}

// IsComplete returns true when a job should be removed from the scheduler.
func (pp *PeriodicProcess) IsComplete(ctx context.Context) bool {
	return false
}

// Equal returns true if another job matches it. Used to cancel jobs.
func (pp *PeriodicProcess) Equal(other Job) bool {
	otherPP, ok := other.(*PeriodicProcess)
	if !ok {
		return false
	}
	return pp.name == otherPP.name
}
