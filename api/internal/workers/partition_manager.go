package workers

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PartitionManager struct {
	db     *pgxpool.Pool
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewPartitionManager(db *pgxpool.Pool) *PartitionManager {
	return &PartitionManager{
		db:     db,
		stopCh: make(chan struct{}),
	}
}

func (pm *PartitionManager) Start(ctx context.Context) error {
	pm.wg.Add(1)
	go pm.run(ctx)
	log.Println("Partition manager started")
	return nil
}

func (pm *PartitionManager) Stop() {
	close(pm.stopCh)
	pm.wg.Wait()
	log.Println("Partition manager stopped")
}

func (pm *PartitionManager) run(ctx context.Context) {
	defer pm.wg.Done()

	// Run daily at 2 AM
	now := time.Now()
	nextRun := time.Date(now.Year(), now.Month(), now.Day()+1, 2, 0, 0, 0, now.Location())
	if now.After(nextRun) {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	timer := time.NewTimer(nextRun.Sub(now))
	defer timer.Stop()

	for {
		select {
		case <-pm.stopCh:
			return
		case <-ctx.Done():
			return
		case <-timer.C:
			pm.managePartitions()
			// Schedule next run
			timer.Reset(24 * time.Hour)
		}
	}
}

func (pm *PartitionManager) managePartitions() {
	ctx := context.Background()

	// Create future partitions (7 days ahead)
	if err := pm.createFuturePartitions(ctx); err != nil {
		log.Printf("Failed to create future partitions: %v", err)
	}

	// Drop old partitions (beyond retention period)
	if err := pm.dropOldPartitions(ctx); err != nil {
		log.Printf("Failed to drop old partitions: %v", err)
	}
}

func (pm *PartitionManager) createFuturePartitions(ctx context.Context) error {
	startDate := time.Now().AddDate(0, 0, 1) // Tomorrow
	endDate := startDate.AddDate(0, 0, 7)    // 7 days ahead

	current := startDate
	for current.Before(endDate) {
		partitionName := fmt.Sprintf("telemetry_y%sm%sd%s",
			current.Format("2006"), current.Format("01"), current.Format("02"))
		partitionStart := current.Format("2006-01-02")

		_, err := pm.db.Exec(ctx, fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s PARTITION OF telemetry
			FOR VALUES FROM ('%s') TO ('%s')`,
			partitionName, partitionStart, current.AddDate(0, 0, 1).Format("2006-01-02")))

		if err != nil {
			return fmt.Errorf("failed to create partition %s: %w", partitionName, err)
		}

		current = current.AddDate(0, 0, 1)
	}

	log.Println("Created future telemetry partitions")
	return nil
}

func (pm *PartitionManager) dropOldPartitions(ctx context.Context) error {
	retentionDays := 30
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	// Query for partitions older than retention period using pg_inherits
	rows, err := pm.db.Query(ctx, `
		SELECT inhrelid::regclass::text as partition_name
		FROM pg_inherits
		JOIN pg_class parent ON pg_inherits.inhparent = parent.oid
		JOIN pg_class child ON pg_inherits.inhrelid = child.oid
		WHERE parent.relname = 'telemetry'
		  AND child.relname LIKE 'telemetry_y%sm%sd%'
		  AND substring(child.relname from 'telemetry_y(\d{4})m(\d{2})d(\d{2})')::date < $1`,
		cutoffDate.Format("2006-01-02"))
	if err != nil {
		return err
	}
	defer rows.Close()

	var partitionsToDrop []string
	for rows.Next() {
		var partitionName string
		if err := rows.Scan(&partitionName); err != nil {
			return err
		}
		partitionsToDrop = append(partitionsToDrop, partitionName)
	}

	// Drop old partitions
	for _, partition := range partitionsToDrop {
		_, err := pm.db.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", partition))
		if err != nil {
			log.Printf("Failed to drop partition %s: %v", partition, err)
			continue
		}
		log.Printf("Dropped old partition: %s", partition)
	}

	if len(partitionsToDrop) > 0 {
		log.Printf("Dropped %d old partitions", len(partitionsToDrop))
	}

	return nil
}