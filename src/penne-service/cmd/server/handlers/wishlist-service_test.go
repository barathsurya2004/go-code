package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.uber.org/cadence/client"
	"go.uber.org/zap"
)

type mockWishlistRepo struct {
	createItemFn          func(item *core.WishlistItem, tx *sql.Tx) (uuid.UUID, error)
	getItemByIDFn         func(id uuid.UUID) (*core.WishlistItem, error)
	getItemsByUserFn      func(userUUID uuid.UUID) ([]*core.WishlistItem, error)
	getActiveItemsByUserFn func(userUUID uuid.UUID) ([]*core.WishlistItem, error)
	updateItemFn          func(item *core.WishlistItem, tx *sql.Tx) error
	deleteItemFn          func(id uuid.UUID) error
	createAllocFn         func(alloc *core.WishlistAllocation, tx *sql.Tx) (uuid.UUID, error)
	getAllocsByItemFn     func(itemID uuid.UUID) ([]*core.WishlistAllocation, error)
	getAllocsByUserFn     func(userUUID uuid.UUID) ([]*core.WishlistAllocation, error)
	updateAllocFn         func(alloc *core.WishlistAllocation, tx *sql.Tx) error
	deleteAllocFn         func(id uuid.UUID) error
}

func (m *mockWishlistRepo) CreateWishlistItem(item *core.WishlistItem, tx *sql.Tx) (uuid.UUID, error) {
	if m.createItemFn != nil {
		return m.createItemFn(item, tx)
	}
	return uuid.New(), nil
}
func (m *mockWishlistRepo) GetWishlistItemByID(id uuid.UUID) (*core.WishlistItem, error) {
	if m.getItemByIDFn != nil {
		return m.getItemByIDFn(id)
	}
	return &core.WishlistItem{ID: id, Title: "Test Item", TargetAmountE5: 10000000}, nil
}
func (m *mockWishlistRepo) GetWishlistItemsByUserUUID(userUUID uuid.UUID) ([]*core.WishlistItem, error) {
	if m.getItemsByUserFn != nil {
		return m.getItemsByUserFn(userUUID)
	}
	return []*core.WishlistItem{}, nil
}
func (m *mockWishlistRepo) GetActiveWishlistItemsByUserUUID(userUUID uuid.UUID) ([]*core.WishlistItem, error) {
	if m.getActiveItemsByUserFn != nil {
		return m.getActiveItemsByUserFn(userUUID)
	}
	return []*core.WishlistItem{}, nil
}
func (m *mockWishlistRepo) UpdateWishlistItem(item *core.WishlistItem, tx *sql.Tx) error {
	if m.updateItemFn != nil {
		return m.updateItemFn(item, tx)
	}
	return nil
}
func (m *mockWishlistRepo) DeleteWishlistItem(id uuid.UUID) error {
	if m.deleteItemFn != nil {
		return m.deleteItemFn(id)
	}
	return nil
}
func (m *mockWishlistRepo) CreateWishlistAllocation(alloc *core.WishlistAllocation, tx *sql.Tx) (uuid.UUID, error) {
	if m.createAllocFn != nil {
		return m.createAllocFn(alloc, tx)
	}
	return uuid.New(), nil
}
func (m *mockWishlistRepo) GetWishlistAllocationsByItemID(itemID uuid.UUID) ([]*core.WishlistAllocation, error) {
	if m.getAllocsByItemFn != nil {
		return m.getAllocsByItemFn(itemID)
	}
	return []*core.WishlistAllocation{}, nil
}
func (m *mockWishlistRepo) GetWishlistAllocationsByUserUUID(userUUID uuid.UUID) ([]*core.WishlistAllocation, error) {
	if m.getAllocsByUserFn != nil {
		return m.getAllocsByUserFn(userUUID)
	}
	return []*core.WishlistAllocation{}, nil
}
func (m *mockWishlistRepo) UpdateWishlistAllocation(alloc *core.WishlistAllocation, tx *sql.Tx) error {
	if m.updateAllocFn != nil {
		return m.updateAllocFn(alloc, tx)
	}
	return nil
}
func (m *mockWishlistRepo) DeleteWishlistAllocation(id uuid.UUID) error {
	if m.deleteAllocFn != nil {
		return m.deleteAllocFn(id)
	}
	return nil
}

func TestWishlistServiceHandler(t *testing.T) {
	logger := zap.NewNop()
	testUserUUID := uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	testItemID := uuid.MustParse("b0000000-0000-0000-0000-000000000001")

	t.Run("CreateWishlistItem - Success", func(t *testing.T) {
		db, _, _ := sqlmock.New()
		defer db.Close()

		wRepo := &mockWishlistRepo{
			createItemFn: func(item *core.WishlistItem, tx *sql.Tx) (uuid.UUID, error) {
				assert.Equal(t, testUserUUID, item.UserUUID)
				assert.Equal(t, "MacBook Pro", item.Title)
				return testItemID, nil
			},
		}
		repos := core.RepoContainer{
			Wishlist: wRepo,
		}

		handler := NewWishlistServiceHandler(repos, logger, db, nil)

		body := `{"title":"MacBook Pro","target_amount_e5":200000000,"priority":5,"urgency":4}`
		req := httptest.NewRequest(http.MethodPost, "/wishlist", bytes.NewBufferString(body))
		ctx := context.WithValue(req.Context(), "user_uuid", testUserUUID)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.CreateWishlistItem(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var resp core.WishlistItem
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, testItemID, resp.ID)
	})

	t.Run("CreateWishlistItem - Missing User UUID", func(t *testing.T) {
		db, _, _ := sqlmock.New()
		defer db.Close()
		handler := NewWishlistServiceHandler(core.RepoContainer{}, logger, db, nil)

		req := httptest.NewRequest(http.MethodPost, "/wishlist", bytes.NewBufferString(`{}`))
		rec := httptest.NewRecorder()

		handler.CreateWishlistItem(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("CreateWishlistItem - Invalid JSON", func(t *testing.T) {
		db, _, _ := sqlmock.New()
		defer db.Close()
		handler := NewWishlistServiceHandler(core.RepoContainer{}, logger, db, nil)

		req := httptest.NewRequest(http.MethodPost, "/wishlist", bytes.NewBufferString(`{invalid json`))
		ctx := context.WithValue(req.Context(), "user_uuid", testUserUUID)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.CreateWishlistItem(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("CreateWishlistItem - Repo Error", func(t *testing.T) {
		db, _, _ := sqlmock.New()
		defer db.Close()
		wRepo := &mockWishlistRepo{
			createItemFn: func(item *core.WishlistItem, tx *sql.Tx) (uuid.UUID, error) {
				return uuid.Nil, errors.New("database insert failed")
			},
		}
		handler := NewWishlistServiceHandler(core.RepoContainer{Wishlist: wRepo}, logger, db, nil)

		body := `{"title":"Phone","target_amount_e5":50000000}`
		req := httptest.NewRequest(http.MethodPost, "/wishlist", bytes.NewBufferString(body))
		ctx := context.WithValue(req.Context(), "user_uuid", testUserUUID)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.CreateWishlistItem(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("GetWishlistItems - Success", func(t *testing.T) {
		db, mockDB, _ := sqlmock.New()
		defer db.Close()

		mockDB.ExpectQuery("SELECT COALESCE").WillReturnRows(
			sqlmock.NewRows([]string{"coalesce"}).AddRow(20000000),
		)

		uRepo := &mockUserRepo{
			getUserByUUIDFn: func(id uuid.UUID) (*core.User, error) {
				return &core.User{UUID: id, MonthlyBudgetE5: 100000000, SalaryDay: 1}, nil
			},
		}
		wRepo := &mockWishlistRepo{
			getItemsByUserFn: func(userUUID uuid.UUID) ([]*core.WishlistItem, error) {
				return []*core.WishlistItem{
					{ID: testItemID, UserUUID: userUUID, Title: "Phone", TargetAmountE5: 80000000, SavedAmountE5: 0, Priority: 5, Urgency: 5, Status: "active"},
				}, nil
			},
		}

		handler := NewWishlistServiceHandler(core.RepoContainer{User: uRepo, Wishlist: wRepo}, logger, db, nil)

		req := httptest.NewRequest(http.MethodGet, "/wishlists", nil)
		ctx := context.WithValue(req.Context(), "user_uuid", testUserUUID)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.GetWishlistItems(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp core.WishlistForecastSummary
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, int64(100000000), resp.MonthlyBudgetE5)
		assert.Equal(t, int64(20000000), resp.CycleExpensesE5)
		assert.Equal(t, int64(80000000), resp.ProjectedSurplusE5)
		assert.Len(t, resp.Items, 1)
	})

	t.Run("GetWishlistItemByID - Success", func(t *testing.T) {
		db, _, _ := sqlmock.New()
		defer db.Close()

		wRepo := &mockWishlistRepo{
			getItemByIDFn: func(id uuid.UUID) (*core.WishlistItem, error) {
				return &core.WishlistItem{ID: id, Title: "Camera", TargetAmountE5: 50000000}, nil
			},
			getAllocsByItemFn: func(itemID uuid.UUID) ([]*core.WishlistAllocation, error) {
				return []*core.WishlistAllocation{
					{ID: uuid.New(), WishlistItemID: itemID, AmountE5: 10000000},
				}, nil
			},
		}

		handler := NewWishlistServiceHandler(core.RepoContainer{Wishlist: wRepo}, logger, db, nil)

		req := httptest.NewRequest(http.MethodGet, "/wishlist?id="+testItemID.String(), nil)
		rec := httptest.NewRecorder()

		handler.GetWishlistItemByID(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp struct {
			Item        *core.WishlistItem         `json:"item"`
			Allocations []*core.WishlistAllocation `json:"allocations"`
		}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "Camera", resp.Item.Title)
		assert.Len(t, resp.Allocations, 1)
	})

	t.Run("GetWishlistItemByID - From Mux Vars", func(t *testing.T) {
		db, _, _ := sqlmock.New()
		defer db.Close()

		wRepo := &mockWishlistRepo{
			getItemByIDFn: func(id uuid.UUID) (*core.WishlistItem, error) {
				return &core.WishlistItem{ID: id, Title: "Trip", TargetAmountE5: 50000000}, nil
			},
		}

		handler := NewWishlistServiceHandler(core.RepoContainer{Wishlist: wRepo}, logger, db, nil)

		req := httptest.NewRequest(http.MethodGet, "/wishlist/"+testItemID.String(), nil)
		req = mux.SetURLVars(req, map[string]string{"id": testItemID.String()})
		rec := httptest.NewRecorder()

		handler.GetWishlistItemByID(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("GetWishlistItemByID - Not Found", func(t *testing.T) {
		db, _, _ := sqlmock.New()
		defer db.Close()

		wRepo := &mockWishlistRepo{
			getItemByIDFn: func(id uuid.UUID) (*core.WishlistItem, error) {
				return nil, errors.New("not found")
			},
		}

		handler := NewWishlistServiceHandler(core.RepoContainer{Wishlist: wRepo}, logger, db, nil)

		req := httptest.NewRequest(http.MethodGet, "/wishlist?id="+testItemID.String(), nil)
		rec := httptest.NewRecorder()

		handler.GetWishlistItemByID(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("UpdateWishlistItem - Success", func(t *testing.T) {
		db, _, _ := sqlmock.New()
		defer db.Close()

		wRepo := &mockWishlistRepo{
			updateItemFn: func(item *core.WishlistItem, tx *sql.Tx) error {
				assert.Equal(t, testItemID, item.ID)
				assert.Equal(t, "Updated Title", item.Title)
				return nil
			},
		}

		handler := NewWishlistServiceHandler(core.RepoContainer{Wishlist: wRepo}, logger, db, nil)

		body := `{"id":"` + testItemID.String() + `","title":"Updated Title","target_amount_e5":100000}`
		req := httptest.NewRequest(http.MethodPut, "/wishlist", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()

		handler.UpdateWishlistItem(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("DeleteWishlistItem - Success", func(t *testing.T) {
		db, _, _ := sqlmock.New()
		defer db.Close()

		wRepo := &mockWishlistRepo{
			deleteItemFn: func(id uuid.UUID) error {
				assert.Equal(t, testItemID, id)
				return nil
			},
		}

		handler := NewWishlistServiceHandler(core.RepoContainer{Wishlist: wRepo}, logger, db, nil)

		req := httptest.NewRequest(http.MethodDelete, "/wishlist?id="+testItemID.String(), nil)
		rec := httptest.NewRecorder()

		handler.DeleteWishlistItem(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("GetForecast - Success", func(t *testing.T) {
		db, mockDB, _ := sqlmock.New()
		defer db.Close()

		mockDB.ExpectQuery("SELECT COALESCE").WillReturnRows(
			sqlmock.NewRows([]string{"coalesce"}).AddRow(10000000),
		)

		uRepo := &mockUserRepo{
			getUserByUUIDFn: func(id uuid.UUID) (*core.User, error) {
				return &core.User{UUID: id, MonthlyBudgetE5: 60000000, SalaryDay: 25}, nil
			},
		}
		wRepo := &mockWishlistRepo{
			getItemsByUserFn: func(userUUID uuid.UUID) ([]*core.WishlistItem, error) {
				return []*core.WishlistItem{}, nil
			},
		}

		handler := NewWishlistServiceHandler(core.RepoContainer{User: uRepo, Wishlist: wRepo}, logger, db, nil)

		req := httptest.NewRequest(http.MethodGet, "/wishlist/forecast", nil)
		ctx := context.WithValue(req.Context(), "user_uuid", testUserUUID)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.GetForecast(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("DistributeSurplus - Direct Engine Execution (No Cadence Client)", func(t *testing.T) {
		db, mockDB, _ := sqlmock.New()
		defer db.Close()

		// CalculateCycleSurplus query
		mockDB.ExpectQuery("SELECT COALESCE").WillReturnRows(
			sqlmock.NewRows([]string{"coalesce"}).AddRow(15000000),
		)
		// DB transaction begin/commit for ApplySurplusDistribution
		mockDB.ExpectBegin()
		mockDB.ExpectCommit()

		uRepo := &mockUserRepo{
			getUserByUUIDFn: func(id uuid.UUID) (*core.User, error) {
				return &core.User{UUID: id, MonthlyBudgetE5: 50000000, SalaryDay: 1}, nil
			},
		}
		wRepo := &mockWishlistRepo{
			getActiveItemsByUserFn: func(userUUID uuid.UUID) ([]*core.WishlistItem, error) {
				return []*core.WishlistItem{
					{
						ID:             testItemID,
						UserUUID:       userUUID,
						Title:          "Trip",
						TargetAmountE5: 100000000,
						SavedAmountE5:  0,
						Priority:       5,
						Urgency:        5,
						Status:         "active",
					},
				}, nil
			},
			createAllocFn: func(alloc *core.WishlistAllocation, tx *sql.Tx) (uuid.UUID, error) {
				return uuid.New(), nil
			},
			updateItemFn: func(item *core.WishlistItem, tx *sql.Tx) error {
				return nil
			},
		}

		handler := NewWishlistServiceHandler(core.RepoContainer{User: uRepo, Wishlist: wRepo}, logger, db, nil)

		req := httptest.NewRequest(http.MethodPost, "/wishlist/distribute", nil)
		ctx := context.WithValue(req.Context(), "user_uuid", testUserUUID)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.DistributeSurplus(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "Surplus distributed successfully", resp["message"])
	})

	t.Run("DistributeSurplus - Cadence Client Execution", func(t *testing.T) {
		db, _, _ := sqlmock.New()
		defer db.Close()

		simResults := []*core.ItemAllocationSimulation{
			{
				ItemID:          testItemID,
				ItemTitle:       "Gadget",
				AllocatedE5:     20000000,
				NewSavedE5:      20000000,
				TargetAmountE5:  50000000,
			},
		}

		cadenceClient := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				assert.Equal(t, "SettleWishlistWorkflow", workflow)
				return &mockWorkflowRun{
					getFn: func(ctx context.Context, valuePtr interface{}) error {
						p, ok := valuePtr.(*[]*core.ItemAllocationSimulation)
						if ok {
							*p = simResults
						}
						return nil
					},
				}, nil
			},
		}

		handler := NewWishlistServiceHandler(core.RepoContainer{}, logger, db, cadenceClient)

		req := httptest.NewRequest(http.MethodPost, "/wishlist/distribute", nil)
		ctx := context.WithValue(req.Context(), "user_uuid", testUserUUID)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.DistributeSurplus(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("UpdateBudgetSettings - Success", func(t *testing.T) {
		db, _, _ := sqlmock.New()
		defer db.Close()

		uRepo := &mockUserRepo{
			updateBudgetSettingsFn: func(userUUID uuid.UUID, monthlyBudgetE5 int64, salaryDay int) error {
				assert.Equal(t, testUserUUID, userUUID)
				assert.Equal(t, int64(80000000), monthlyBudgetE5)
				assert.Equal(t, 25, salaryDay)
				return nil
			},
		}

		handler := NewWishlistServiceHandler(core.RepoContainer{User: uRepo}, logger, db, nil)

		body := `{"monthly_budget_e5":80000000,"salary_day":25}`
		req := httptest.NewRequest(http.MethodPut, "/user/budget-settings", bytes.NewBufferString(body))
		ctx := context.WithValue(req.Context(), "user_uuid", testUserUUID)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.UpdateBudgetSettings(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("UpdateBudgetSettings - Fallback for SalaryDay <= 0", func(t *testing.T) {
		db, _, _ := sqlmock.New()
		defer db.Close()

		uRepo := &mockUserRepo{
			updateBudgetSettingsFn: func(userUUID uuid.UUID, monthlyBudgetE5 int64, salaryDay int) error {
				assert.Equal(t, 1, salaryDay)
				return nil
			},
		}

		handler := NewWishlistServiceHandler(core.RepoContainer{User: uRepo}, logger, db, nil)

		body := `{"monthly_budget_e5":80000000,"salary_day":0}`
		req := httptest.NewRequest(http.MethodPut, "/user/budget-settings", bytes.NewBufferString(body))
		ctx := context.WithValue(req.Context(), "user_uuid", testUserUUID)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.UpdateBudgetSettings(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("GetWishlistItems - Error Cases", func(t *testing.T) {
		db, mock, _ := sqlmock.New()
		defer db.Close()

		handler := NewWishlistServiceHandler(core.RepoContainer{}, logger, db, nil)

		// 1. Missing user UUID
		req1 := httptest.NewRequest(http.MethodGet, "/wishlists", nil)
		rec1 := httptest.NewRecorder()
		handler.GetWishlistItems(rec1, req1)
		assert.Equal(t, http.StatusBadRequest, rec1.Code)

		// 2. User not found
		errUserRepo := &mockUserRepo{
			getUserByUUIDFn: func(id uuid.UUID) (*core.User, error) {
				return nil, errors.New("user not found")
			},
		}
		h2 := NewWishlistServiceHandler(core.RepoContainer{User: errUserRepo}, logger, db, nil)
		req2 := httptest.NewRequest(http.MethodGet, "/wishlists", nil)
		req2 = req2.WithContext(context.WithValue(req2.Context(), "user_uuid", testUserUUID))
		rec2 := httptest.NewRecorder()
		h2.GetWishlistItems(rec2, req2)
		assert.Equal(t, http.StatusNotFound, rec2.Code)

		// 3. Cycle surplus error
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(testUserUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("db error"))
		goodUserRepo := &mockUserRepo{
			getUserByUUIDFn: func(id uuid.UUID) (*core.User, error) {
				return &core.User{UUID: testUserUUID, MonthlyBudgetE5: 1000, SalaryDay: 1}, nil
			},
		}
		h3 := NewWishlistServiceHandler(core.RepoContainer{User: goodUserRepo}, logger, db, nil)
		req3 := httptest.NewRequest(http.MethodGet, "/wishlists", nil)
		req3 = req3.WithContext(context.WithValue(req3.Context(), "user_uuid", testUserUUID))
		rec3 := httptest.NewRecorder()
		h3.GetWishlistItems(rec3, req3)
		assert.Equal(t, http.StatusInternalServerError, rec3.Code)

		// 4. Wishlist items fetch error
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(testUserUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(0)))
		errWishRepo := &mockWishlistRepo{
			getItemsByUserFn: func(userUUID uuid.UUID) ([]*core.WishlistItem, error) {
				return nil, errors.New("fetch error")
			},
		}
		h4 := NewWishlistServiceHandler(core.RepoContainer{User: goodUserRepo, Wishlist: errWishRepo}, logger, db, nil)
		req4 := httptest.NewRequest(http.MethodGet, "/wishlists", nil)
		req4 = req4.WithContext(context.WithValue(req4.Context(), "user_uuid", testUserUUID))
		rec4 := httptest.NewRecorder()
		h4.GetWishlistItems(rec4, req4)
		assert.Equal(t, http.StatusInternalServerError, rec4.Code)
	})

	t.Run("GetWishlistItemByID - Error Cases", func(t *testing.T) {
		db, _, _ := sqlmock.New()
		defer db.Close()

		handler := NewWishlistServiceHandler(core.RepoContainer{}, logger, db, nil)

		// Item not found
		errWishRepo := &mockWishlistRepo{
			getItemByIDFn: func(id uuid.UUID) (*core.WishlistItem, error) {
				return nil, errors.New("not found")
			},
		}
		h2 := NewWishlistServiceHandler(core.RepoContainer{Wishlist: errWishRepo}, logger, db, nil)
		req := httptest.NewRequest(http.MethodGet, "/wishlist?id="+testItemID.String(), nil)
		rec := httptest.NewRecorder()
		h2.GetWishlistItemByID(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)

		// Allocations error fallback to empty slice
		goodWishRepo := &mockWishlistRepo{
			getItemByIDFn: func(id uuid.UUID) (*core.WishlistItem, error) {
				return &core.WishlistItem{ID: id, Title: "Item"}, nil
			},
			getAllocsByItemFn: func(itemID uuid.UUID) ([]*core.WishlistAllocation, error) {
				return nil, errors.New("alloc error")
			},
		}
		h3 := NewWishlistServiceHandler(core.RepoContainer{Wishlist: goodWishRepo}, logger, db, nil)
		req3 := httptest.NewRequest(http.MethodGet, "/wishlist?id="+testItemID.String(), nil)
		rec3 := httptest.NewRecorder()
		h3.GetWishlistItemByID(rec3, req3)
		assert.Equal(t, http.StatusOK, rec3.Code)
		_ = handler
	})

	t.Run("UpdateWishlistItem - Error and ID in Query Cases", func(t *testing.T) {
		db, _, _ := sqlmock.New()
		defer db.Close()

		// Update error from repo
		errRepo := &mockWishlistRepo{
			updateItemFn: func(item *core.WishlistItem, tx *sql.Tx) error {
				return errors.New("failed update")
			},
		}
		h := NewWishlistServiceHandler(core.RepoContainer{Wishlist: errRepo}, logger, db, nil)
		req := httptest.NewRequest(http.MethodPut, "/wishlist", bytes.NewBufferString(`{"id":"`+testItemID.String()+`","title":"Updated"}`))
		rec := httptest.NewRecorder()
		h.UpdateWishlistItem(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		// ID supplied via query string
		successRepo := &mockWishlistRepo{
			updateItemFn: func(item *core.WishlistItem, tx *sql.Tx) error {
				assert.Equal(t, testItemID, item.ID)
				return nil
			},
		}
		h2 := NewWishlistServiceHandler(core.RepoContainer{Wishlist: successRepo}, logger, db, nil)
		req2 := httptest.NewRequest(http.MethodPut, "/wishlist?id="+testItemID.String(), bytes.NewBufferString(`{"title":"Updated"}`))
		rec2 := httptest.NewRecorder()
		h2.UpdateWishlistItem(rec2, req2)
		assert.Equal(t, http.StatusOK, rec2.Code)
	})

	t.Run("DeleteWishlistItem - Error Cases", func(t *testing.T) {
		db, _, _ := sqlmock.New()
		defer db.Close()

		// Missing ID
		h := NewWishlistServiceHandler(core.RepoContainer{}, logger, db, nil)
		req := httptest.NewRequest(http.MethodDelete, "/wishlist", nil)
		rec := httptest.NewRecorder()
		h.DeleteWishlistItem(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		// Delete error from repo
		errRepo := &mockWishlistRepo{
			deleteItemFn: func(id uuid.UUID) error {
				return errors.New("delete error")
			},
		}
		h2 := NewWishlistServiceHandler(core.RepoContainer{Wishlist: errRepo}, logger, db, nil)
		req2 := httptest.NewRequest(http.MethodDelete, "/wishlist?id="+testItemID.String(), nil)
		rec2 := httptest.NewRecorder()
		h2.DeleteWishlistItem(rec2, req2)
		assert.Equal(t, http.StatusInternalServerError, rec2.Code)
	})

	t.Run("GetForecast - Error Cases", func(t *testing.T) {
		db, mock, _ := sqlmock.New()
		defer db.Close()

		handler := NewWishlistServiceHandler(core.RepoContainer{}, logger, db, nil)

		// Missing user UUID
		req1 := httptest.NewRequest(http.MethodGet, "/wishlist/forecast", nil)
		rec1 := httptest.NewRecorder()
		handler.GetForecast(rec1, req1)
		assert.Equal(t, http.StatusBadRequest, rec1.Code)

		// User not found
		errUserRepo := &mockUserRepo{
			getUserByUUIDFn: func(id uuid.UUID) (*core.User, error) {
				return nil, errors.New("user not found")
			},
		}
		h2 := NewWishlistServiceHandler(core.RepoContainer{User: errUserRepo}, logger, db, nil)
		req2 := httptest.NewRequest(http.MethodGet, "/wishlist/forecast", nil)
		req2 = req2.WithContext(context.WithValue(req2.Context(), "user_uuid", testUserUUID))
		rec2 := httptest.NewRecorder()
		h2.GetForecast(rec2, req2)
		assert.Equal(t, http.StatusNotFound, rec2.Code)

		// Surplus calculation error
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(testUserUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("db error"))
		goodUserRepo := &mockUserRepo{
			getUserByUUIDFn: func(id uuid.UUID) (*core.User, error) {
				return &core.User{UUID: testUserUUID, MonthlyBudgetE5: 1000, SalaryDay: 1}, nil
			},
		}
		h3 := NewWishlistServiceHandler(core.RepoContainer{User: goodUserRepo}, logger, db, nil)
		req3 := httptest.NewRequest(http.MethodGet, "/wishlist/forecast", nil)
		req3 = req3.WithContext(context.WithValue(req3.Context(), "user_uuid", testUserUUID))
		rec3 := httptest.NewRecorder()
		h3.GetForecast(rec3, req3)
		assert.Equal(t, http.StatusInternalServerError, rec3.Code)

		// Items fetch error
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(testUserUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(0)))
		errWishRepo := &mockWishlistRepo{
			getItemsByUserFn: func(userUUID uuid.UUID) ([]*core.WishlistItem, error) {
				return nil, errors.New("items error")
			},
		}
		h4 := NewWishlistServiceHandler(core.RepoContainer{User: goodUserRepo, Wishlist: errWishRepo}, logger, db, nil)
		req4 := httptest.NewRequest(http.MethodGet, "/wishlist/forecast", nil)
		req4 = req4.WithContext(context.WithValue(req4.Context(), "user_uuid", testUserUUID))
		rec4 := httptest.NewRecorder()
		h4.GetForecast(rec4, req4)
		assert.Equal(t, http.StatusInternalServerError, rec4.Code)
	})

	t.Run("DistributeSurplus - Error Cases", func(t *testing.T) {
		db, mock, _ := sqlmock.New()
		defer db.Close()

		handler := NewWishlistServiceHandler(core.RepoContainer{}, logger, db, nil)

		// 1. Missing user UUID
		req1 := httptest.NewRequest(http.MethodPost, "/wishlist/distribute", nil)
		rec1 := httptest.NewRecorder()
		handler.DistributeSurplus(rec1, req1)
		assert.Equal(t, http.StatusBadRequest, rec1.Code)

		// 2. User not found (engine returns error, handler returns 400)
		errUserRepo := &mockUserRepo{
			getUserByUUIDFn: func(id uuid.UUID) (*core.User, error) {
				return nil, errors.New("user not found")
			},
		}
		h2 := NewWishlistServiceHandler(core.RepoContainer{User: errUserRepo}, logger, db, nil)
		req2 := httptest.NewRequest(http.MethodPost, "/wishlist/distribute", nil)
		req2 = req2.WithContext(context.WithValue(req2.Context(), "user_uuid", testUserUUID))
		rec2 := httptest.NewRecorder()
		h2.DistributeSurplus(rec2, req2)
		assert.Equal(t, http.StatusBadRequest, rec2.Code)

		// 3. Active items error (engine returns error, handler returns 400)
		goodUserRepo := &mockUserRepo{
			getUserByUUIDFn: func(id uuid.UUID) (*core.User, error) {
				return &core.User{UUID: testUserUUID, MonthlyBudgetE5: 1000, SalaryDay: 1}, nil
			},
		}
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(testUserUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(0)))
		errWishRepo := &mockWishlistRepo{
			getActiveItemsByUserFn: func(userUUID uuid.UUID) ([]*core.WishlistItem, error) {
				return nil, errors.New("items error")
			},
		}
		h3 := NewWishlistServiceHandler(core.RepoContainer{User: goodUserRepo, Wishlist: errWishRepo}, logger, db, nil)
		req3 := httptest.NewRequest(http.MethodPost, "/wishlist/distribute", nil)
		req3 = req3.WithContext(context.WithValue(req3.Context(), "user_uuid", testUserUUID))
		rec3 := httptest.NewRecorder()
		h3.DistributeSurplus(rec3, req3)
		assert.Equal(t, http.StatusBadRequest, rec3.Code)

		// 4. No active items
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(testUserUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(0)))
		emptyWishRepo := &mockWishlistRepo{
			getActiveItemsByUserFn: func(userUUID uuid.UUID) ([]*core.WishlistItem, error) {
				return []*core.WishlistItem{}, nil
			},
		}
		h4 := NewWishlistServiceHandler(core.RepoContainer{User: goodUserRepo, Wishlist: emptyWishRepo}, logger, db, nil)
		req4 := httptest.NewRequest(http.MethodPost, "/wishlist/distribute", nil)
		req4 = req4.WithContext(context.WithValue(req4.Context(), "user_uuid", testUserUUID))
		rec4 := httptest.NewRecorder()
		h4.DistributeSurplus(rec4, req4)
		assert.Equal(t, http.StatusBadRequest, rec4.Code)

		// 5. Surplus <= 0
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(testUserUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(2000))) // expenses > budget (1000)
		hasItemsRepo := &mockWishlistRepo{
			getActiveItemsByUserFn: func(userUUID uuid.UUID) ([]*core.WishlistItem, error) {
				return []*core.WishlistItem{{ID: testItemID, Title: "Item"}}, nil
			},
		}
		h5 := NewWishlistServiceHandler(core.RepoContainer{User: goodUserRepo, Wishlist: hasItemsRepo}, logger, db, nil)
		req5 := httptest.NewRequest(http.MethodPost, "/wishlist/distribute", nil)
		req5 = req5.WithContext(context.WithValue(req5.Context(), "user_uuid", testUserUUID))
		rec5 := httptest.NewRecorder()
		h5.DistributeSurplus(rec5, req5)
		assert.Equal(t, http.StatusBadRequest, rec5.Code)

		// 6. Direct application error (fallback when cadence is nil)
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(testUserUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(0))) // surplus = 1000
		mock.ExpectBegin().WillReturnError(errors.New("tx error"))
		h6 := NewWishlistServiceHandler(core.RepoContainer{User: goodUserRepo, Wishlist: hasItemsRepo}, logger, db, nil)
		req6 := httptest.NewRequest(http.MethodPost, "/wishlist/distribute", nil)
		req6 = req6.WithContext(context.WithValue(req6.Context(), "user_uuid", testUserUUID))
		rec6 := httptest.NewRecorder()
		h6.DistributeSurplus(rec6, req6)
		assert.Equal(t, http.StatusBadRequest, rec6.Code)

		// 7. Cadence execute workflow error
		errCadenceClient := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return nil, errors.New("cadence workflow failed")
			},
		}
		h7 := NewWishlistServiceHandler(core.RepoContainer{User: goodUserRepo, Wishlist: hasItemsRepo}, logger, db, errCadenceClient)
		req7 := httptest.NewRequest(http.MethodPost, "/wishlist/distribute", nil)
		req7 = req7.WithContext(context.WithValue(req7.Context(), "user_uuid", testUserUUID))
		rec7 := httptest.NewRecorder()
		h7.DistributeSurplus(rec7, req7)
		assert.Equal(t, http.StatusInternalServerError, rec7.Code)

		// 8. Cadence workflow get error
		errGetCadenceClient := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return &mockWorkflowRun{
					getFn: func(ctx context.Context, valuePtr interface{}) error {
						return errors.New("get result failed")
					},
				}, nil
			},
		}
		h8 := NewWishlistServiceHandler(core.RepoContainer{User: goodUserRepo, Wishlist: hasItemsRepo}, logger, db, errGetCadenceClient)
		req8 := httptest.NewRequest(http.MethodPost, "/wishlist/distribute", nil)
		req8 = req8.WithContext(context.WithValue(req8.Context(), "user_uuid", testUserUUID))
		rec8 := httptest.NewRecorder()
		h8.DistributeSurplus(rec8, req8)
		assert.Equal(t, http.StatusInternalServerError, rec8.Code)
	})

	t.Run("UpdateBudgetSettings - Error Cases", func(t *testing.T) {
		db, _, _ := sqlmock.New()
		defer db.Close()

		handler := NewWishlistServiceHandler(core.RepoContainer{}, logger, db, nil)

		// Missing user UUID
		req1 := httptest.NewRequest(http.MethodPut, "/user/budget-settings", nil)
		rec1 := httptest.NewRecorder()
		handler.UpdateBudgetSettings(rec1, req1)
		assert.Equal(t, http.StatusBadRequest, rec1.Code)

		// Invalid JSON
		req2 := httptest.NewRequest(http.MethodPut, "/user/budget-settings", bytes.NewBufferString(`invalid-json`))
		req2 = req2.WithContext(context.WithValue(req2.Context(), "user_uuid", testUserUUID))
		rec2 := httptest.NewRecorder()
		handler.UpdateBudgetSettings(rec2, req2)
		assert.Equal(t, http.StatusBadRequest, rec2.Code)

		// Repo update error
		errRepo := &mockUserRepo{
			updateBudgetSettingsFn: func(userUUID uuid.UUID, monthlyBudgetE5 int64, salaryDay int) error {
				return errors.New("db error")
			},
		}
		h5 := NewWishlistServiceHandler(core.RepoContainer{User: errRepo}, logger, db, nil)
		req5 := httptest.NewRequest(http.MethodPut, "/user/budget-settings", bytes.NewBufferString(`{"monthly_budget_e5": 100, "salary_day": 20}`))
		req5 = req5.WithContext(context.WithValue(req5.Context(), "user_uuid", testUserUUID))
		rec5 := httptest.NewRecorder()
		h5.UpdateBudgetSettings(rec5, req5)
		assert.Equal(t, http.StatusBadRequest, rec5.Code)
	})
}

