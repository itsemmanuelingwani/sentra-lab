package test

import (
	"context"
	"sync"
	"time"
)

type ParallelExecutor struct {
	maxWorkers int
	failFast   bool
}

func NewParallelExecutor(maxWorkers int, failFast bool) *ParallelExecutor {
	return &ParallelExecutor{
		maxWorkers: maxWorkers,
		failFast:   failFast,
	}
}

func (pe *ParallelExecutor) Execute(ctx context.Context, tasks []Task) ([]TaskResult, error) {
	results := make([]TaskResult, len(tasks))
	resultsMu := sync.Mutex{}

	semaphore := make(chan struct{}, pe.maxWorkers)
	var wg sync.WaitGroup
	stopChan := make(chan struct{})
	firstError := make(chan error, 1)

	for i, task := range tasks {
		wg.Add(1)

		go func(idx int, t Task) {
			defer wg.Done()

			select {
			case <-stopChan:
				resultsMu.Lock()
				results[idx] = TaskResult{
					Index:  idx,
					Status: "skipped",
					Task:   t,
				}
				resultsMu.Unlock()
				return
			case semaphore <- struct{}{}:
			}

			defer func() { <-semaphore }()

			startTime := time.Now()
			err := t.Execute(ctx)
			duration := time.Since(startTime)

			result := TaskResult{
				Index:    idx,
				Task:     t,
				Duration: duration,
				Error:    err,
			}

			if err != nil {
				result.Status = "failed"

				if pe.failFast {
					select {
					case firstError <- err:
						close(stopChan)
					default:
					}
				}
			} else {
				result.Status = "success"
			}

			resultsMu.Lock()
			results[idx] = result
			resultsMu.Unlock()

		}(i, task)
	}

	wg.Wait()

	select {
	case err := <-firstError:
		return results, err
	default:
		return results, nil
	}
}

type Task interface {
	Execute(ctx context.Context) error
	Name() string
}

type TaskResult struct {
	Index    int
	Task     Task
	Status   string
	Duration time.Duration
	Error    error
}

type ScenarioTask struct {
	scenarioPath string
	runner       *Runner
	progressFn   func(string, string, float64)
}

func NewScenarioTask(scenarioPath string, runner *Runner, progressFn func(string, string, float64)) *ScenarioTask {
	return &ScenarioTask{
		scenarioPath: scenarioPath,
		runner:       runner,
		progressFn:   progressFn,
	}
}

func (st *ScenarioTask) Execute(ctx context.Context) error {
	_, err := st.runner.runScenario(ctx, st.scenarioPath, st.progressFn)
	return err
}

func (st *ScenarioTask) Name() string {
	return st.scenarioPath
}

type WorkerPool struct {
	workers    int
	taskQueue  chan Task
	resultChan chan TaskResult
	wg         sync.WaitGroup
}

func NewWorkerPool(workers int) *WorkerPool {
	return &WorkerPool{
		workers:    workers,
		taskQueue:  make(chan Task, workers*2),
		resultChan: make(chan TaskResult, workers*2),
	}
}

func (wp *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(ctx, i)
	}
}

func (wp *WorkerPool) worker(ctx context.Context, id int) {
	defer wp.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-wp.taskQueue:
			if !ok {
				return
			}

			startTime := time.Now()
			err := task.Execute(ctx)
			duration := time.Since(startTime)

			result := TaskResult{
				Task:     task,
				Duration: duration,
				Error:    err,
			}

			if err != nil {
				result.Status = "failed"
			} else {
				result.Status = "success"
			}

			select {
			case wp.resultChan <- result:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (wp *WorkerPool) Submit(task Task) {
	wp.taskQueue <- task
}

func (wp *WorkerPool) Close() {
	close(wp.taskQueue)
}

func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
	close(wp.resultChan)
}

func (wp *WorkerPool) Results() <-chan TaskResult {
	return wp.resultChan
}
