package core

import "time"

type PlannedSet struct {
	ID     string  `json:"id"`
	Type   string  `json:"type"` // warmup, working, drop, failure
	Reps   int     `json:"reps"`
	Weight float64 `json:"weight"`
	RPE    *float64 `json:"rpe,omitempty"`
}

type RoutineBlock struct {
	ID         string       `json:"id"`
	ExerciseID string       `json:"exerciseId"`
	Sets       []PlannedSet `json:"sets"`
	Rest       int          `json:"rest"`
	SupersetID *string      `json:"supersetId,omitempty"`
	Note       *string      `json:"note,omitempty"`
	TargetReps *string      `json:"targetReps,omitempty"`
}

type Routine struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Focus         string         `json:"focus"`
	Color         string         `json:"color"`
	Blocks        []RoutineBlock `json:"blocks"`
	EstMinutes    int            `json:"estMinutes"`
	LastPerformed *string        `json:"lastPerformed,omitempty"`
	WarmupNote    *string        `json:"warmupNote,omitempty"`
	NutritionNote *string        `json:"nutritionNote,omitempty"`
}

type LoggedSet struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Reps        int      `json:"reps"`
	Weight      float64  `json:"weight"`
	RPE         *float64 `json:"rpe,omitempty"`
	Done        bool     `json:"done"`
	IsPR        *bool    `json:"isPR,omitempty"`
	CompletedAt *int64   `json:"completedAt,omitempty"`
}

type SessionBlock struct {
	ID         string      `json:"id"`
	ExerciseID string      `json:"exerciseId"`
	Sets       []LoggedSet `json:"sets"`
	Rest       int         `json:"rest"`
	SupersetID *string     `json:"supersetId,omitempty"`
	Note       *string     `json:"note,omitempty"`
	TargetReps *string     `json:"targetReps,omitempty"`
	BarWeight  *float64    `json:"barWeight,omitempty"`
}

type WorkoutSession struct {
	ID        string         `json:"id"`
	RoutineID *string        `json:"routineId,omitempty"`
	Name      string         `json:"name"`
	Color     string         `json:"color"`
	StartedAt int64          `json:"startedAt"`
	EndedAt   *int64         `json:"endedAt,omitempty"`
	Blocks    []SessionBlock `json:"blocks"`
	RPE       *float64       `json:"rpe,omitempty"`
	Mood      *int           `json:"mood,omitempty"`
	Notes     *string        `json:"notes,omitempty"`
}

type PlanDay struct {
	Weekday   int     `json:"weekday"`
	RoutineID *string `json:"routineId"`
}

type Settings struct {
	Unit          string  `json:"unit"`
	BarWeight     float64 `json:"barWeight"`
	DefaultRest   int     `json:"defaultRest"`
	Haptics       bool    `json:"haptics"`
	Sound         bool    `json:"sound"`
	AutoStartRest bool    `json:"autoStartRest"`
	WeeklyGoal    int     `json:"weeklyGoal"`
	Bodyweight    float64 `json:"bodyweight"`
	Name          string  `json:"name"`
}

type SyncPayload struct {
	UserUUID     string           `json:"user_uuid"`
	ClientTime   int64            `json:"client_time"`
	Routines     []Routine        `json:"routines,omitempty"`
	Plan         []PlanDay        `json:"plan,omitempty"`
	Sessions     []WorkoutSession `json:"sessions,omitempty"`
	Settings     *Settings        `json:"settings,omitempty"`
	SkippedDates []string         `json:"skipped_dates,omitempty"`
}

type SyncResponse struct {
	Status         string `json:"status"`
	Message        string `json:"message"`
	SyncedAt       int64  `json:"synced_at"`
	SessionsSynced int    `json:"sessions_synced"`
	RoutinesSynced int    `json:"routines_synced"`
}

type PulseUserStats struct {
	UserUUID       string    `json:"user_uuid"`
	TotalSessions  int       `json:"total_sessions"`
	TotalRoutines  int       `json:"total_routines"`
	LastSyncedAt   time.Time `json:"last_synced_at"`
}

type PulseUser struct {
	ID           string    `json:"id"`
	UserUUID     string    `json:"user_uuid"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
	UserUUID string `json:"user_uuid,omitempty"`
}

type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token    string     `json:"token"`
	UserUUID string     `json:"user_uuid"`
	User     *PulseUser `json:"user"`
	Message  string     `json:"message,omitempty"`
}
