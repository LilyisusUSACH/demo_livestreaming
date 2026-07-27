package cache

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheService struct {
	client     *redis.Client
	useMemory  bool
	memStorage map[string]string
	memBytes   map[string][]byte
	memLists   map[string][][]byte
	memMu      sync.RWMutex
}

func NewCacheService(redisAddr string) *CacheService {
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     "",
		DB:           0,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cs := &CacheService{
		client:     rdb,
		memStorage: make(map[string]string),
		memBytes:   make(map[string][]byte),
		memLists:   make(map[string][][]byte),
	}

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Redis no disponible en %s (%v). Usando In-Memory Cache Fallback.", redisAddr, err)
		cs.useMemory = true
	} else {
		log.Printf("Conexion con Redis exitosa en: %s", redisAddr)
	}

	return cs
}

// -----------------------------------------------------------------------------
// Redis Per-Channel Live Chat Storage & Pub/Sub Infrastructure
// -----------------------------------------------------------------------------

func (cs *CacheService) SaveChatMessage(ctx context.Context, channelID string, msgBytes []byte) error {
	key := fmt.Sprintf("chat:history:%s", channelID)
	if cs.useMemory {
		cs.memMu.Lock()
		defer cs.memMu.Unlock()
		list := cs.memLists[key]
		list = append(list, msgBytes)
		if len(list) > 50 {
			list = list[len(list)-50:]
		}
		cs.memLists[key] = list
		return nil
	}

	pipe := cs.client.Pipeline()
	pipe.RPush(ctx, key, msgBytes)
	pipe.LTrim(ctx, key, -50, -1) // Keep last 50 messages in Redis
	pipe.Expire(ctx, key, 7*24*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

func (cs *CacheService) GetChatHistory(ctx context.Context, channelID string) ([][]byte, error) {
	key := fmt.Sprintf("chat:history:%s", channelID)
	if cs.useMemory {
		cs.memMu.RLock()
		defer cs.memMu.RUnlock()
		list := cs.memLists[key]
		res := make([][]byte, len(list))
		copy(res, list)
		return res, nil
	}

	cmd := cs.client.LRange(ctx, key, 0, -1)
	resStrings, err := cmd.Result()
	if err != nil {
		return nil, err
	}

	res := make([][]byte, len(resStrings))
	for i, s := range resStrings {
		res[i] = []byte(s)
	}
	return res, nil
}

func (cs *CacheService) PublishChatMessage(ctx context.Context, channelID string, msgBytes []byte) error {
	topic := fmt.Sprintf("chat:pubsub:%s", channelID)
	if cs.useMemory {
		return nil
	}
	return cs.client.Publish(ctx, topic, msgBytes).Err()
}

// -----------------------------------------------------------------------------
// HLS Segment & Playlist Redis Caching Infrastructure
// -----------------------------------------------------------------------------

func (cs *CacheService) CacheSegment(ctx context.Context, filename string, data []byte, duration time.Duration) error {
	key := fmt.Sprintf("hls:segment:%s", filename)
	if cs.useMemory {
		cs.memMu.Lock()
		cs.memBytes[key] = data
		cs.memMu.Unlock()
		return nil
	}
	return cs.client.Set(ctx, key, data, duration).Err()
}

func (cs *CacheService) GetCachedSegment(ctx context.Context, filename string) ([]byte, bool) {
	key := fmt.Sprintf("hls:segment:%s", filename)
	if cs.useMemory {
		cs.memMu.RLock()
		data, exists := cs.memBytes[key]
		cs.memMu.RUnlock()
		return data, exists
	}
	data, err := cs.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	return data, true
}

func (cs *CacheService) CachePlaylist(ctx context.Context, key string, playlistStr string, duration time.Duration) error {
	redisKey := fmt.Sprintf("hls:playlist:%s", key)
	if cs.useMemory {
		cs.memMu.Lock()
		cs.memStorage[redisKey] = playlistStr
		cs.memMu.Unlock()
		return nil
	}
	return cs.client.Set(ctx, redisKey, playlistStr, duration).Err()
}

func (cs *CacheService) GetCachedPlaylist(ctx context.Context, key string) (string, bool) {
	redisKey := fmt.Sprintf("hls:playlist:%s", key)
	if cs.useMemory {
		cs.memMu.RLock()
		val, exists := cs.memStorage[redisKey]
		cs.memMu.RUnlock()
		return val, exists
	}
	val, err := cs.client.Get(ctx, redisKey).Result()
	if err != nil {
		return "", false
	}
	return val, true
}

// -----------------------------------------------------------------------------
// Auth Token & Session Tracking Infrastructure
// -----------------------------------------------------------------------------

func (cs *CacheService) StoreRefreshToken(ctx context.Context, userID, tokenID string, duration time.Duration) error {
	key := fmt.Sprintf("refresh_token:%s:%s", userID, tokenID)
	if cs.useMemory {
		cs.memMu.Lock()
		cs.memStorage[key] = userID
		cs.memMu.Unlock()
		return nil
	}
	return cs.client.Set(ctx, key, userID, duration).Err()
}

func (cs *CacheService) IsRefreshTokenValid(ctx context.Context, userID, tokenID string) bool {
	key := fmt.Sprintf("refresh_token:%s:%s", userID, tokenID)
	if cs.useMemory {
		cs.memMu.RLock()
		_, exists := cs.memStorage[key]
		cs.memMu.RUnlock()
		return exists
	}
	val, err := cs.client.Get(ctx, key).Result()
	return err == nil && val == userID
}

func (cs *CacheService) RevokeRefreshToken(ctx context.Context, userID, tokenID string) error {
	key := fmt.Sprintf("refresh_token:%s:%s", userID, tokenID)
	if cs.useMemory {
		cs.memMu.Lock()
		delete(cs.memStorage, key)
		cs.memMu.Unlock()
		return nil
	}
	return cs.client.Del(ctx, key).Err()
}

func (cs *CacheService) RevokeAllUserTokens(ctx context.Context, userID string) error {
	pattern := fmt.Sprintf("refresh_token:%s:*", userID)
	if cs.useMemory {
		cs.memMu.Lock()
		prefix := fmt.Sprintf("refresh_token:%s:", userID)
		for k := range cs.memStorage {
			if strings.HasPrefix(k, prefix) {
				delete(cs.memStorage, k)
			}
		}
		cs.memMu.Unlock()
		return nil
	}

	iter := cs.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		cs.client.Del(ctx, iter.Val())
	}
	return iter.Err()
}

func (cs *CacheService) CountActiveUserSessions(ctx context.Context, userID string) int {
	prefix := fmt.Sprintf("refresh_token:%s:", userID)
	if cs.useMemory {
		cs.memMu.RLock()
		defer cs.memMu.RUnlock()
		count := 0
		for k := range cs.memStorage {
			if strings.HasPrefix(k, prefix) {
				count++
			}
		}
		return count
	}

	pattern := prefix + "*"
	iter := cs.client.Scan(ctx, 0, pattern, 0).Iterator()
	count := 0
	for iter.Next(ctx) {
		count++
	}
	return count
}

func (cs *CacheService) GetAllActiveSessions(ctx context.Context) (map[string]int, error) {
	result := make(map[string]int)

	if cs.useMemory {
		cs.memMu.RLock()
		defer cs.memMu.RUnlock()
		for k := range cs.memStorage {
			parts := strings.Split(k, ":")
			if len(parts) == 3 {
				uID := parts[1]
				result[uID]++
			}
		}
		return result, nil
	}

	iter := cs.client.Scan(ctx, 0, "refresh_token:*", 0).Iterator()
	for iter.Next(ctx) {
		parts := strings.Split(iter.Val(), ":")
		if len(parts) == 3 {
			uID := parts[1]
			result[uID]++
		}
	}
	return result, iter.Err()
}
