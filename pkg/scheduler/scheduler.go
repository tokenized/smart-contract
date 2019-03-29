package scheduler

import (
	"context"
	"sync"
	"time"
)

// Scheduler provides the ability to schedule tasks to run at when they are ready.
type Scheduler struct {
	jobs          []Job
	lock          sync.Mutex
	isRunning     bool
	stopRequested bool
}

// Job provides an interface that tells Scheduler when and how to run the job.
type Job interface {
	// IsReady returns true when a job should be executed.
	IsReady(ctx context.Context) bool

	// Run executes the job.
	Run(ctx context.Context)

	// IsComplete returns true when a job should be removed from the scheduler.
	IsComplete(ctx context.Context) bool
}

// ScheduleJob adds a job to the scheduler.
func (sch *Scheduler) ScheduleJob(ctx context.Context, job Job) error {
	sch.lock.Lock()
	defer sch.lock.Unlock()
	sch.jobs = append(sch.jobs, job)
	return nil
}

// Run monitors jobs and runs them when they are ready.
func (sch *Scheduler) Run(ctx context.Context) error {
	sch.lock.Lock()
	sch.isRunning = true
	for !sch.stopRequested {
		// Check and run jobs
		for i, job := range sch.jobs {
			if job.IsReady(ctx) {
				job.Run(ctx)
				if job.IsComplete(ctx) {
					sch.jobs = append(sch.jobs[:i], sch.jobs[i+1:]...)
					break // Modified list being iterated
				}
			}
		}

		// Unlock for sleep
		sch.lock.Unlock()
		time.Sleep(500 * time.Millisecond)
		sch.lock.Lock()
	}
	sch.isRunning = false
	sch.lock.Unlock()
	return nil
}

// stillRunning returns true if the scheduler is still running.
func (sch *Scheduler) stillRunning() bool {
	sch.lock.Lock()
	defer sch.lock.Unlock()
	return sch.isRunning
}

// Stop requests Run finish and waits for it to finish.
func (sch *Scheduler) Stop(ctx context.Context) error {
	sch.lock.Lock()
	sch.stopRequested = true
	sch.lock.Unlock()

	for sch.stillRunning() {
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}
