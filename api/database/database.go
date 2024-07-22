package database

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
)

var Ctx = context.Background()

// CreateClient creates a new Redis client with the specified database number
// Database 0: User authorization
// Database 1: Rate limiting
// Database 2: Shortened URL storage
func CreateClient(dbNo int) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("DB_ADDR"),
		Password: os.Getenv("DB_PASS"),
		DB:       dbNo,
	})
	return rdb
}

/*func TestInitialize(dbNo int) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("DB_ADDR"),
		Password: os.Getenv("DB_PASS"),
		DB:       dbNo,
	})
	// Test authenticate
	rdb.HSet(Ctx, "Coffeetooth", "Password", "Test", "Secret", "Bigbadwolf")
	rdb.HExpire(Ctx, "Coffeetooth", 24*time.Hour, "Secret")
	rdb.SAdd(Ctx, "users:public", "Coffeetooth")
}
*/
