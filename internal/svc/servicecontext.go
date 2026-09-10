// Package svc assembles shared dependencies; it contains no HTTP handlers or SQL.
package svc

import (
	"database/sql"
	"net/http"
	"sync"
	"time"

	"github.com/chemanyu/adn-report-ai/internal/config"
	"github.com/chemanyu/adn-report-ai/internal/dingtalk"
	"github.com/chemanyu/adn-report-ai/internal/model"
)

type ServiceContext struct {
	Config       config.Config
	AuthModel    model.AuthModel
	UploadModel  model.UploadModel
	DingTalk     *dingtalk.Client
	LoginLimiter *LoginLimiter
	ImportSlots  chan struct{}
}

func NewServiceContext(c config.Config, db *sql.DB) *ServiceContext {
	return &ServiceContext{
		Config: c, AuthModel: model.NewAuthModel(db), UploadModel: model.NewUploadModel(db),
		DingTalk: &dingtalk.Client{HTTPClient: &http.Client{Timeout: 15 * time.Second}}, LoginLimiter: &LoginLimiter{}, ImportSlots: make(chan struct{}, 2),
	}
}

// LoginLimiter is shared by requests so invalid credentials cannot reset the limit.
type LoginLimiter struct {
	mu       sync.Mutex
	attempts int
	reset    time.Time
}

func (l *LoginLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if now.After(l.reset) {
		l.attempts = 0
		l.reset = now.Add(time.Minute)
	}
	l.attempts++
	return l.attempts <= 20
}
