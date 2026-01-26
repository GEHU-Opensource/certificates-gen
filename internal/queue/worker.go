package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type Job struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt time.Time              `json:"created_at"`
}

type Worker struct {
	client     *redis.Client
	queueName  string
	workerID   string
	processors map[string]JobProcessor
}

type JobProcessor func(ctx context.Context, job Job) error

func NewWorker(client *redis.Client, queueName, workerID string) *Worker {
	return &Worker{
		client:     client,
		queueName:  queueName,
		workerID:   workerID,
		processors: make(map[string]JobProcessor),
	}
}

func (w *Worker) RegisterProcessor(jobType string, processor JobProcessor) {
	w.processors[jobType] = processor
}

func (w *Worker) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := w.processNext(ctx); err != nil {
				time.Sleep(1 * time.Second)
				continue
			}
		}
	}
}

func (w *Worker) processNext(ctx context.Context) error {
	result, err := w.client.BRPop(ctx, 5*time.Second, w.queueName).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to pop from queue: %w", err)
	}

	if len(result) < 2 {
		return nil
	}

	var job Job
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return fmt.Errorf("failed to unmarshal job: %w", err)
	}

	processor, ok := w.processors[job.Type]
	if !ok {
		return fmt.Errorf("unknown job type: %s", job.Type)
	}

	if err := processor(ctx, job); err != nil {
		return fmt.Errorf("job processing failed: %w", err)
	}

	return nil
}

func (w *Worker) Enqueue(ctx context.Context, job Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	if err := w.client.LPush(ctx, w.queueName, data).Err(); err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	return nil
}

func (w *Worker) EnqueueBatch(ctx context.Context, jobs []Job) error {
	pipe := w.client.Pipeline()
	for _, job := range jobs {
		data, err := json.Marshal(job)
		if err != nil {
			continue
		}
		pipe.LPush(ctx, w.queueName, data)
	}
	_, err := pipe.Exec(ctx)
	return err
}
