package worker

import (
	"context"

	"github.com/hibiken/asynq"
)

type TaskDistributor interface {
	DistributeTaskSendEmail(
		ctx context.Context,
		payload *PayloadSendEmail,
		opts ...asynq.Option,
	) error
	Close() error
}

type RedisTaskDistributor struct {
	client *asynq.Client
}

func NewRedisTaskDistributor(redisOpt asynq.RedisClientOpt) TaskDistributor {
	ConfigureRedisLogging()

	client := asynq.NewClient(redisOpt)
	return &RedisTaskDistributor{
		client: client,
	}
}

func (distributor *RedisTaskDistributor) Close() error {
	return distributor.client.Close()
}
