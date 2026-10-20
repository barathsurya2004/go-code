package db

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/barathsurya2004/go-code/pulse-service/internal/core"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidUserUUID    = errors.New("invalid or empty user_uuid")
	ErrEmptyPayload       = errors.New("empty sync payload")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user with this email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

type PulseRepository interface {
	SaveSyncPayload(ctx context.Context, payload *core.SyncPayload) (*core.SyncResponse, error)
	GetUserStats(ctx context.Context, userUUID string) (*core.PulseUserStats, error)
	DeleteUserData(ctx context.Context, userUUID string) error

	// User Management
	CreateUser(ctx context.Context, req *core.CreateUserRequest) (*core.PulseUser, error)
	GetUserByEmail(ctx context.Context, email string) (*core.PulseUser, error)
	GetUserByUUID(ctx context.Context, userUUID string) (*core.PulseUser, error)
	AuthenticateUser(ctx context.Context, email, password string) (*core.PulseUser, error)

	Close() error
}

type MemoryPulseRepository struct {
	mu           sync.RWMutex
	users        map[string]time.Time
	pulseUsers   map[string]*core.PulseUser
	usersByEmail map[string]*core.PulseUser
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
		pulseUsers:   make(map[string]*core.PulseUser),
		usersByEmail: make(map[string]*core.PulseUser),
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

func (m *MemoryPulseRepository) DeleteUserData(ctx context.Context, userUUID string) error {
	if userUUID == "" {
		return ErrInvalidUserUUID
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.users, userUUID)
	delete(m.settings, userUUID)
	delete(m.routines, userUUID)
	delete(m.sessions, userUUID)
	delete(m.plans, userUUID)
	delete(m.skippedDates, userUUID)
	if u, ok := m.pulseUsers[userUUID]; ok {
		delete(m.usersByEmail, u.Email)
		delete(m.pulseUsers, userUUID)
	}
	return nil
}

func (m *MemoryPulseRepository) CreateUser(ctx context.Context, req *core.CreateUserRequest) (*core.PulseUser, error) {
	if req == nil {
		return nil, errors.New("empty create user request")
	}
	if req.Email == "" {
		return nil, errors.New("email is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.usersByEmail[req.Email]; exists {
		return nil, ErrUserAlreadyExists
	}

	uuidStr := req.UserUUID
	if uuidStr == "" {
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		uuidStr = hex.EncodeToString(b)
	}

	var passHash string
	if req.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		passHash = string(hashed)
	}

	now := time.Now()
	user := &core.PulseUser{
		ID:           uuidStr,
		UserUUID:     uuidStr,
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: passHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	m.pulseUsers[uuidStr] = user
	m.usersByEmail[req.Email] = user
	return user, nil
}

func (m *MemoryPulseRepository) GetUserByEmail(ctx context.Context, email string) (*core.PulseUser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, ok := m.usersByEmail[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	copyUser := *user
	return &copyUser, nil
}

func (m *MemoryPulseRepository) GetUserByUUID(ctx context.Context, userUUID string) (*core.PulseUser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, ok := m.pulseUsers[userUUID]
	if !ok {
		return nil, ErrUserNotFound
	}
	copyUser := *user
	return &copyUser, nil
}

func (m *MemoryPulseRepository) AuthenticateUser(ctx context.Context, email, password string) (*core.PulseUser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, ok := m.usersByEmail[email]
	if !ok {
		return nil, ErrInvalidCredentials
	}

	if user.PasswordHash != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
			return nil, ErrInvalidCredentials
		}
	}

	copyUser := *user
	return &copyUser, nil
}

func (m *MemoryPulseRepository) Close() error {
	return nil
}
