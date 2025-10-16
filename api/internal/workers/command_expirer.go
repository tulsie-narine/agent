package workers

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CommandExpirer struct {
	db     *pgxpool.Pool
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewCommandExpirer(db *pgxpool.Pool) *CommandExpirer {
	return &CommandExpirer{
		db:     db,
		stopCh: make(chan struct{}),
	}
}

func (e *CommandExpirer) Start(ctx context.Context) error {
	e.wg.Add(1)
	go e.run(ctx)
	log.Println("Command expirer started")
	return nil
}

func (e *CommandExpirer) Stop() {
	close(e.stopCh)
	e.wg.Wait()
	log.Println("Command expirer stopped")
}

func (e *CommandExpirer) run(ctx context.Context) {
	defer e.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.expireCommands()
		}
	}
}

func (e *CommandExpirer) expireCommands() {
	ctx := context.Background()

	result, err := e.db.Exec(ctx, `
		UPDATE commands
		SET status = 'expired'
		WHERE status = 'pending'
		  AND issued_at + (ttl_seconds || ' seconds')::interval < NOW()`)

	if err != nil {
		log.Printf("Failed to expire commands: %v", err)
		return
	}

	if rowsAffected := result.RowsAffected(); rowsAffected > 0 {
		log.Printf("Expired %d stale commands", rowsAffected)
	}
}