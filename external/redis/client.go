package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type RedisService struct {
	client RedisClientInterface
	logger *zap.Logger
}

// RedisClientInterface defines the methods of *redis.Client that RedisService actually uses.
type RedisClientInterface interface {
	Ping(ctx context.Context) *redis.StatusCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
}

func NewRedisService(addr string, logger *zap.Logger) *RedisService {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		logger.Error("Failed to connect to Redis, caching will be unavailable", zap.String("address", addr), zap.Error(err))
	} else {
		logger.Info("Successfully connected to Redis", zap.String("address", addr))
	}

	return &RedisService{
		client: client,
		logger: logger,
	}
}

// CacheSentMessage caches a sent message ID and its external ID along with the sent time
// for 24 hours
func (r *RedisService) CacheSentMessage(ctx context.Context, messageID, externalMessageID string, sentAt time.Time) error {
	// Key format: sent_messages:<message_id>
	key := fmt.Sprintf("sent_messages:%s", messageID)
	value := fmt.Sprintf("ext_id:%s;sent_at:%s", externalMessageID, sentAt.Format(time.RFC3339))
	expiration := 24 * time.Hour

	if r.client == nil {
		r.logger.Warn("Redis client is not initialized, cannot cache message", zap.String("message_id", messageID))
		return fmt.Errorf("redis client not initialized")
	}

	err := r.client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		r.logger.Error("Failed to cache sent message in Redis",
			zap.String("message_id", messageID),
			zap.String("external_id", externalMessageID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to cache message %s: %w", messageID, err)
	}
	r.logger.Debug("Successfully cached sent message", zap.String("message_id", messageID), zap.String("external_id", externalMessageID))
	return nil
}
