package ratelimit

import (
	"context"
	"golang-swing-trading-signal/internal/config"
	"golang-swing-trading-signal/internal/utils"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"gopkg.in/telebot.v3"
)

type userLimiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

type messageLimiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

type TelegramRateLimiter struct {
	cfg             *config.TelegramConfig
	log             *logrus.Logger
	globalLimiter   *rate.Limiter
	userLimiters    map[int64]*userLimiterEntry
	messageLimiters map[int64]*messageLimiterEntry
	bot             *telebot.Bot
	mu              sync.Mutex
	wg              sync.WaitGroup
}

func NewTelegramRateLimiter(cfg *config.TelegramConfig, log *logrus.Logger, bot *telebot.Bot) *TelegramRateLimiter {

	return &TelegramRateLimiter{
		cfg:             cfg,
		log:             log,
		bot:             bot,
		globalLimiter:   rate.NewLimiter(rate.Limit(cfg.MaxGlobalRequestPerSecond), cfg.MaxGlobalRequestPerSecond), // 30 msg/sec globally
		userLimiters:    make(map[int64]*userLimiterEntry),
		messageLimiters: make(map[int64]*messageLimiterEntry),
		mu:              sync.Mutex{},
		wg:              sync.WaitGroup{},
	}
}

func (t *TelegramRateLimiter) Send(ctx context.Context, c telebot.Context, what interface{}, opts ...interface{}) (*telebot.Message, error) {
	if err := t.checkRateLimit(ctx, c); err != nil {
		return nil, err
	}
	return t.bot.Send(c.Chat(), what, opts...)
}

func (t *TelegramRateLimiter) SendWithoutLimit(ctx context.Context, c telebot.Context, what interface{}, opts ...interface{}) (*telebot.Message, error) {
	return t.bot.Send(c.Chat(), what, opts...)
}

func (t *TelegramRateLimiter) SendWithoutMsg(ctx context.Context, c telebot.Context, what interface{}, opts ...interface{}) error {
	if err := t.checkRateLimit(ctx, c); err != nil {
		return err
	}
	_, err := t.Send(ctx, c, what, opts...)
	if err != nil {
		t.log.WithError(err).Error("Failed to send message")
		return err
	}
	return nil
}

func (t *TelegramRateLimiter) Edit(ctx context.Context, c telebot.Context, msg *telebot.Message, what interface{}, opts ...interface{}) (*telebot.Message, error) {
	if err := t.checkRateLimit(ctx, c); err != nil {
		return nil, err
	}
	return t.bot.Edit(msg, what, opts...)
}

func (t *TelegramRateLimiter) EditWithoutLimit(ctx context.Context, c telebot.Context, msg *telebot.Message, what interface{}, opts ...interface{}) (*telebot.Message, error) {
	return t.bot.Edit(msg, what, opts...)
}

func (t *TelegramRateLimiter) DeleteWithoutLimit(ctx context.Context, c telebot.Context, msg *telebot.Message) error {
	return t.bot.Delete(msg)
}

func (t *TelegramRateLimiter) EditWithoutMsg(ctx context.Context, c telebot.Context, what interface{}, opts ...interface{}) error {
	if err := t.checkRateLimit(ctx, c); err != nil {
		return err
	}
	_, err := t.Edit(ctx, c, c.Message(), what, opts...)
	if err != nil {
		t.log.WithError(err).Error("Failed to edit message")
		return err
	}
	return nil
}

func (t *TelegramRateLimiter) Respond(ctx context.Context, c telebot.Context, resp ...*telebot.CallbackResponse) error {
	if err := t.checkRateLimit(ctx, c); err != nil {
		return err
	}
	return c.Respond(resp...)
}

func (r *TelegramRateLimiter) getUserLimiter(userID int64) *userLimiterEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	if limiter, exists := r.userLimiters[userID]; exists {
		limiter.lastAccess = time.Now()
		return limiter
	}

	// Buat limiter baru untuk user ini: 1 msg/sec
	limiter := rate.NewLimiter(rate.Limit(r.cfg.MaxUserRequestPerSecond), r.cfg.MaxUserRequestPerSecond)
	r.userLimiters[userID] = &userLimiterEntry{
		limiter:    limiter,
		lastAccess: time.Now(),
	}
	return r.userLimiters[userID]
}

func (r *TelegramRateLimiter) getMessageLimiter(userID int64) *messageLimiterEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	if limiter, exists := r.messageLimiters[userID]; exists {
		limiter.lastAccess = time.Now()
		return limiter
	}

	// Buat limiter baru untuk user ini: 1 msg/sec
	limiter := rate.NewLimiter(rate.Limit(r.cfg.MaxEditMessagePerSecond), r.cfg.MaxEditMessagePerSecond)
	r.messageLimiters[userID] = &messageLimiterEntry{
		limiter:    limiter,
		lastAccess: time.Now(),
	}
	return r.messageLimiters[userID]
}

func (r *TelegramRateLimiter) checkRateLimit(ctx context.Context, c telebot.Context) error {
	userLimiter := r.getUserLimiter(c.Sender().ID)
	messageLimiter := r.getMessageLimiter(c.Chat().ID)

	if err := messageLimiter.limiter.Wait(ctx); err != nil {
		r.log.WithError(err).Error("Failed to wait for message rate limit")
		return err
	}
	if err := r.globalLimiter.Wait(ctx); err != nil {
		r.log.WithError(err).Error("Failed to wait for global rate limit")
		return err
	}
	if err := userLimiter.limiter.Wait(ctx); err != nil {
		r.log.WithError(err).Error("Failed to wait for user rate limit")
		return err
	}
	return nil
}
func (r *TelegramRateLimiter) StartCleanupExpired(ctx context.Context) {
	r.wg.Add(1)
	utils.SafeGo(func() {
		defer r.wg.Done()
		ticker := time.NewTicker(r.cfg.RateLimitCleanupDuration)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				r.log.Info("Received signal to stop Telegram rate limiter cleanup expired")
				return
			case <-ticker.C:
				r.mu.Lock()
				now := time.Now()
				for userID, entry := range r.userLimiters {
					if now.Sub(entry.lastAccess) > r.cfg.RatelimitExpireDuration {
						delete(r.userLimiters, userID)
					}
				}
				r.mu.Unlock()
			}
		}
	})
}

func (r *TelegramRateLimiter) StopCleanupExpired() {
	r.wg.Wait()
	r.log.Info("Telegram rate limiter stopped")
}
