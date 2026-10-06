package worker

import (
	"context"
	"time"
)

func (w *Worker) CleanupRoutine(ctx context.Context) {
	const processedKillsTTL = 14 * 24 * time.Hour

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("[Cleanup] Routine stopped")
			return
		default:
		}

		if err := w.subStore.CleanupProcessedKills(processedKillsTTL); err != nil {
			w.logger.Error("[Cleanup] Cleanup processed kills err", "error", err)
		}

		select {
		case <-ctx.Done():
			w.logger.Info("[Cleanup] Routine stopped")
			return
		case <-time.After(6 * time.Hour):
		}
	}
}
