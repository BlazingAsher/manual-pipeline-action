package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func startCleanup(ctx context.Context, pool *pgxpool.Pool, interval, maxAge time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				tag, err := pool.Exec(ctx,
					`DELETE FROM questions WHERE created_at < NOW() - $1::interval`,
					maxAge.String(),
				)
				if err != nil {
					log.Printf("cleanup error: %v", err)
				} else if tag.RowsAffected() > 0 {
					log.Printf("cleanup: deleted %d old question(s)", tag.RowsAffected())
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
