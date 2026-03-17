package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/persist"
	rediswatcher "github.com/casbin/redis-watcher/v2"
	"github.com/mrhumster/identity-service/config"
	"github.com/redis/go-redis/v9"
)

type PermissionClient interface {
	CheckPermission(ctx context.Context, userID, resource, action string) (bool, error)
	AddPolicy(ctx context.Context, userID, resource, action string) (bool, error)
	RemovePolicy(ctx context.Context, userID, resource, action string) (bool, error)
	AddPolicyIfNotExists(ctx context.Context, userID, resource, action string) (bool, error)
	Close() error
}

type PermissionService struct {
	enforcer *casbin.Enforcer
	watcher  persist.Watcher
	mu       sync.RWMutex
}

func NewPermissionService(e *casbin.Enforcer, cfg config.Redis) (*PermissionService, error) {
	ps := &PermissionService{
		enforcer: e,
	}

	slog.Info("Redis conn password ", "length", len(cfg.Password))

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       0,
	})

	w, err := rediswatcher.NewWatcher(cfg.Addr, rediswatcher.WatcherOptions{
		SubClient: redisClient,
		PubClient: redisClient,
		Channel:   "/casbin",
	})
	if err != nil {
		return nil, fmt.Errorf("error create csbin watcher: %w", err)
	}
	if err := e.SetWatcher(w); err != nil {
		return nil, fmt.Errorf("error setting watcher for casbin enforcer: %w", err)
	}
	w.SetUpdateCallback(func(msg string) {
		slog.Info("🔄 Casbin watcher signal received", "msg", msg)
		ps.mu.Lock()
		err := ps.enforcer.LoadPolicy()
		ps.mu.Unlock()
		if err != nil {
			slog.Error("❌ Failed to reload Casbin policies", "error", err)
		} else {
			slog.Info("✅ Casbin policies reloaded successfully")
		}
	})
	return ps, nil
}

func (p *PermissionService) AddPolicyIfNotExists(sub, obj, act string) (bool, error) {
	hasPolicy, err := p.enforcer.HasPolicy(sub, obj, act)
	if err != nil {
		return false, err
	}

	if !hasPolicy {
		success, err := p.AddPolicy(sub, obj, act)
		return success, err
	}
	return true, nil
}

func (p *PermissionService) CheckPermission(subj, obj, act string) (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enforcer.Enforce(subj, obj, act)
}

func (p *PermissionService) AddPolicy(sub, obj, act string) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.enforcer.AddPolicy(sub, obj, act)
}

func (p *PermissionService) RemovePolicy(sub, obj, act string) (bool, error) {
	return p.enforcer.RemovePolicy(sub, obj, act)
}

func (p *PermissionService) Close() error {
	if p.watcher != nil {
		p.watcher.Close()
	}
	return nil
}
