package worker

import (
	"context"
	"time"

	"LazyCatBot/internal/news"
)

// AnnounceOnce posts the current news.md to every known welcome channel once.
// Idempotency lives in bot_meta keyed by the content hash, so no command, cron
// or per-guild delivery table is needed.
func (w *Worker) AnnounceOnce(ctx context.Context) {
	text := news.Content()
	if text == "" {
		return
	}

	key := "announce:" + news.Version()
	if seen, err := w.subStore.MetaGet(key); err != nil {
		w.logger.Error("[Announce] Read meta err", "error", err)
	} else if seen != "" {
		return
	}

	channels, err := w.subStore.GetAllWelcomeChannels()
	if err != nil {
		w.logger.Error("[Announce] Load welcome channels err", "error", err)
		return
	}
	// On a fresh database the welcome channels are populated by GuildCreate
	// events right after the session opens; give them a moment before giving up.
	if len(channels) == 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second):
		}
		channels, err = w.subStore.GetAllWelcomeChannels()
		if err != nil {
			w.logger.Error("[Announce] Load welcome channels err", "error", err)
			return
		}
	}

	sent := 0
	for _, ch := range channels {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := w.reporter.SendAnnouncement(ch, text); err != nil {
			w.logger.Warn("[Announce] Send err", "error", err, "channel", ch)
			continue
		}
		sent++
		time.Sleep(750 * time.Millisecond)
	}

	if sent == 0 {
		return
	}
	if err := w.subStore.MetaSet(key, time.Now().UTC().Format(time.RFC3339)); err != nil {
		w.logger.Error("[Announce] Save meta err", "error", err)
	}
	w.logger.Info("[Announce] Sent", "channels", sent, "version", news.Version())
}
