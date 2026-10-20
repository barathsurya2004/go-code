package db

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/barathsurya2004/go-code/pulse-service/internal/core"
	"go.uber.org/zap"
)

var (
	ErrInvalidUserUUID = errors.New("invalid or empty user_uuid")
	ErrEmptyPayload    = errors.New("empty sync payload")
)

type PulseRepository interface {
	SaveSyncPayload(ctx context.Context, payload *core.SyncPayload) (*core.SyncResponse, error)
	GetUserStats(ctx context.Context, userUUID string) (*core.PulseUserStats, error)
	Close() error
}

type MemoryPulseRepository struct {
	mu           sync.RWMutex
	users        map[string]time.Time
	settings     map[string]*core.Settings
	routines     map[string]map[string]*core.Routine        // userUUID -> routineID -> Routine
	sessions     map[string]map[string]*core.WorkoutSession // userUUID -> sessionID -> Session
	plans        map[string]map[int]*core.PlanDay           // userUUID -> weekday -> PlanDay
	skippedDates map[string]map[string]bool                 // userUUID -> dateKey -> true
	log          *zap.Logger
}

func NewMemoryPulseRepository(log *zap.Logger) *MemoryPulseRepository {
	return &MemoryPulseRepository{
		users:        make(map[string]time.Time),
		settings:     make(map[string]*core.Settings),
		routines:     make(map[string]map[string]*core.Routine),
		sessions:     make(map[string]map[string]*core.WorkoutSession),
		plans:        make(map[string]map[int]*core.PlanDay),
		skippedDates: make(map[string]map[string]bool),
		log:          log,
	}
}

func (m *MemoryPulseRepository) SaveSyncPayload(ctx context.Context, payload *core.SyncPayload) (*core.SyncResponse, error) {
	if payload == nil {
		return nil, ErrEmptyPayload
	}
	if payload.UserUUID == "" {
		return nil, ErrInvalidUserUUID
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	m.users[payload.UserUUID] = now

	if payload.Settings != nil {
		copySettings := *payload.Settings
		m.settings[payload.UserUUID] = &copySettings
	}

	if _, exists := m.routines[payload.UserUUID]; !exists {
		m.routines[payload.UserUUID] = make(map[string]*core.Routine)
	}
	for _, r := range payload.Routines {
		copyRoutine := r
		m.routines[payload.UserUUID][r.ID] = &copyRoutine
	}

	if _, exists := m.plans[payload.UserUUID]; !exists {
		m.plans[payload.UserUUID] = make(map[int]*core.PlanDay)
	}
	for _, p := range payload.Plan {
		copyPlan := p
		m.plans[payload.UserUUID][p.Weekday] = &copyPlan
	}

	if _, exists := m.sessions[payload.UserUUID]; !exists {
		m.sessions[payload.UserUUID] = make(map[string]*core.WorkoutSession)
	}
	for _, s := range payload.Sessions {
		copySession := s
		m.sessions[payload.UserUUID][s.ID] = &copySession
	}

	if _, exists := m.skippedDates[payload.UserUUID]; !exists {
		m.skippedDates[payload.UserUUID] = make(map[string]bool)
	}
	for _, d := range payload.SkippedDates {
		m.skippedDates[payload.UserUUID][d] = true
	}

	return &core.SyncResponse{
		Status:         "ok",
		Message:        "Push sync completed successfully",
		SyncedAt:       now.UnixMilli(),
		SessionsSynced: len(payload.Sessions),
		RoutinesSynced: len(payload.Routines),
	}, nil
}

func (m *MemoryPulseRepository) GetUserStats(ctx context.Context, userUUID string) (*core.PulseUserStats, error) {
	if userUUID == "" {
		return nil, ErrInvalidUserUUID
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	lastSync := m.users[userUUID]
	totalSessions := 0
	if userSessions, ok := m.sessions[userUUID]; ok {
		totalSessions = len(userSessions)
	}

	totalRoutines := 0
	if userRoutines, ok := m.routines[userUUID]; ok {
		totalRoutines = len(userRoutines)
	}

	return &core.PulseUserStats{
		UserUUID:      userUUID,
		TotalSessions: totalSessions,
		TotalRoutines: totalRoutines,
		LastSyncedAt:  lastSync,
	}, nil
}

func (m *MemoryPulseRepository) Close() error {
	return nil
}
