package scheduler

import (
	"context"
	"time"
)

type PeriodicTaskInterface interface {
	Run(context.Context)
}

// PeriodicTask is a Scheduler Task that runs a process at a specified frequency.
type PeriodicTask struct {
	name      string
	process   PeriodicTaskInterface
	frequency time.Duration
	next      time.Time
}

func NewPeriodicTask(name string, process PeriodicTaskInterface, frequency time.Duration) *PeriodicTask {
	return &PeriodicTask{
		name:      name,
		process:   process,
		frequency: frequency,
		next:      time.Now().Add(frequency),
	}
}

// IsReady returns true when a job should be executed.
func (pp *PeriodicTask) IsReady(ctx context.Context) bool {
	return time.Now().After(pp.next)
}

// Run executes the job.
func (pp *PeriodicTask) Run(ctx context.Context) {
	// Schedule next time
	pp.next = time.Now().Add(pp.frequency)

	// Run process
	pp.process.Run(ctx)
}

// IsComplete returns true when a job should be removed from the scheduler.
func (pp *PeriodicTask) IsComplete(ctx context.Context) bool {
	return false
}

// Equal returns true if another job matches it. Used to cancel jobs.
func (pp *PeriodicTask) Equal(other Task) bool {
	otherPP, ok := other.(*PeriodicTask)
	if !ok {
		return false
	}
	return pp.name == otherPP.name
}
