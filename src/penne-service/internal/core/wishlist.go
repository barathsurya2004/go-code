package core

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type WishlistItem struct {
	ID             uuid.UUID  `json:"id"`
	UserUUID       uuid.UUID  `json:"user_uuid"`
	Title          string     `json:"title"`
	TargetAmountE5 int64      `json:"target_amount_e5"`
	SavedAmountE5  int64      `json:"saved_amount_e5"`
	Priority       int        `json:"priority"`
	Urgency        int        `json:"urgency"`
	ItemType       string     `json:"item_type"`
	Status         string     `json:"status"`
	TargetDate     *time.Time `json:"target_date,omitempty"`
	Notes          string     `json:"notes,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type WishlistAllocation struct {
	ID             uuid.UUID `json:"id"`
	WishlistItemID uuid.UUID `json:"wishlist_item_id"`
	UserUUID       uuid.UUID `json:"user_uuid"`
	AmountE5       int64     `json:"amount_e5"`
	SourceType     string    `json:"source_type"`
	CycleDate      time.Time `json:"cycle_date"`
	CreatedAt      time.Time `json:"created_at"`
}

type ItemAllocationSimulation struct {
	ItemID          uuid.UUID `json:"item_id"`
	ItemTitle       string    `json:"item_title"`
	AllocatedE5     int64     `json:"allocated_e5"`
	PreviousSavedE5 int64     `json:"previous_saved_e5"`
	NewSavedE5      int64     `json:"new_saved_e5"`
	TargetAmountE5  int64     `json:"target_amount_e5"`
	IsFulfilled     bool      `json:"is_fulfilled"`
	Weight          int       `json:"weight"`
}

type ItemForecast struct {
	ItemID                uuid.UUID  `json:"item_id"`
	ItemTitle             string     `json:"item_title"`
	TargetAmountE5        int64      `json:"target_amount_e5"`
	SavedAmountE5         int64      `json:"saved_amount_e5"`
	RemainingAmountE5     int64      `json:"remaining_amount_e5"`
	Priority              int        `json:"priority"`
	Urgency               int        `json:"urgency"`
	Weight                int        `json:"weight"`
	MonthlyContributionE5 int64      `json:"monthly_contribution_e5"`
	EstimatedMonths       float64    `json:"estimated_months"`
	EstimatedDate         *time.Time `json:"estimated_date,omitempty"`
	ProgressPercentage    float64    `json:"progress_percentage"`
}

type WishlistForecastSummary struct {
	CycleStartDate     time.Time       `json:"cycle_start_date"`
	CycleEndDate       time.Time       `json:"cycle_end_date"`
	MonthlyBudgetE5    int64           `json:"monthly_budget_e5"`
	CycleExpensesE5    int64           `json:"cycle_expenses_e5"`
	ProjectedSurplusE5 int64           `json:"projected_surplus_e5"`
	SavingsRatePercent float64         `json:"savings_rate_percent"`
	Items              []*ItemForecast `json:"items"`
}

type WishlistRepository interface {
	CreateWishlistItem(item *WishlistItem, Tx *sql.Tx) (uuid.UUID, error)
	GetWishlistItemByID(id uuid.UUID) (*WishlistItem, error)
	GetWishlistItemsByUserUUID(userUUID uuid.UUID) ([]*WishlistItem, error)
	GetActiveWishlistItemsByUserUUID(userUUID uuid.UUID) ([]*WishlistItem, error)
	UpdateWishlistItem(item *WishlistItem, Tx *sql.Tx) error
	DeleteWishlistItem(id uuid.UUID) error

	CreateWishlistAllocation(alloc *WishlistAllocation, Tx *sql.Tx) (uuid.UUID, error)
	GetWishlistAllocationsByItemID(itemID uuid.UUID) ([]*WishlistAllocation, error)
	GetWishlistAllocationsByUserUUID(userUUID uuid.UUID) ([]*WishlistAllocation, error)
}
