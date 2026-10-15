package grpcserver

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/barathsurya2004/go-code/nutrition-service/internal/core"
	nutritionv1 "github.com/barathsurya2004/go-code/pkg/proto/nutritionv1"
	"github.com/google/uuid"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type NutritionGRPCServer struct {
	nutritionv1.UnimplementedNutritionServiceServer
	repo   core.NutritionRepository
	worker core.ImageQueueWorker
	log    *zap.Logger
}

func NewNutritionGRPCServer(
	repo core.NutritionRepository,
	worker core.ImageQueueWorker,
	log *zap.Logger,
) *NutritionGRPCServer {
	return &NutritionGRPCServer{
		repo:   repo,
		worker: worker,
		log:    log,
	}
}

func extractUserUUID(ctx context.Context, fallback string) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		vals := md.Get("x-user-uuid")
		if len(vals) > 0 && vals[0] != "" {
			return vals[0]
		}
	}
	if fallback != "" {
		return fallback
	}
	return "00000000-0000-0000-0000-000000000001"
}

func (s *NutritionGRPCServer) CheckHealth(ctx context.Context, req *nutritionv1.HealthCheckRequest) (*nutritionv1.HealthCheckResponse, error) {
	return &nutritionv1.HealthCheckResponse{
		Status:      "SERVING",
		ServiceName: "nutrition-service",
		Timestamp:   time.Now().Unix(),
	}, nil
}

func (s *NutritionGRPCServer) GetGoals(ctx context.Context, req *nutritionv1.GetGoalsRequest) (*nutritionv1.NutritionGoals, error) {
	userUUID := extractUserUUID(ctx, req.GetUserUuid())
	goals, err := s.repo.GetGoals(ctx, userUUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get goals: %v", err)
	}

	return &nutritionv1.NutritionGoals{
		UserUuid:       goals.UserUUID,
		CaloriesTarget: goals.CaloriesTarget,
		ProteinGrams:   goals.ProteinGrams,
		CarbsGrams:     goals.CarbsGrams,
		FatGrams:       goals.FatGrams,
		FiberGrams:     goals.FiberGrams,
		SodiumMg:       goals.SodiumMg,
	}, nil
}

func (s *NutritionGRPCServer) SetGoals(ctx context.Context, req *nutritionv1.SetGoalsRequest) (*nutritionv1.NutritionGoals, error) {
	if req.Goals == nil {
		return nil, status.Errorf(codes.InvalidArgument, "goals payload is required")
	}

	userUUID := extractUserUUID(ctx, req.Goals.GetUserUuid())
	saved, err := s.repo.SetGoals(ctx, &core.NutritionGoals{
		UserUUID:       userUUID,
		CaloriesTarget: req.Goals.GetCaloriesTarget(),
		ProteinGrams:   req.Goals.GetProteinGrams(),
		CarbsGrams:     req.Goals.GetCarbsGrams(),
		FatGrams:       req.Goals.GetFatGrams(),
		FiberGrams:     req.Goals.GetFiberGrams(),
		SodiumMg:       req.Goals.GetSodiumMg(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set goals: %v", err)
	}

	return &nutritionv1.NutritionGoals{
		UserUuid:       saved.UserUUID,
		CaloriesTarget: saved.CaloriesTarget,
		ProteinGrams:   saved.ProteinGrams,
		CarbsGrams:     saved.CarbsGrams,
		FatGrams:       saved.FatGrams,
		FiberGrams:     saved.FiberGrams,
		SodiumMg:       saved.SodiumMg,
	}, nil
}

func (s *NutritionGRPCServer) GetDailyLog(ctx context.Context, req *nutritionv1.GetDailyLogRequest) (*nutritionv1.DailyNutritionSummary, error) {
	userUUID := extractUserUUID(ctx, req.GetUserUuid())
	dateStr := req.GetDate()
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	summary, err := s.repo.GetDailyLog(ctx, userUUID, dateStr)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get daily log: %v", err)
	}

	protoMeals := make([]*nutritionv1.MealLog, 0)
	for _, m := range summary.Meals {
		var protoItems []*nutritionv1.MealItem
		for _, it := range m.Items {
			protoItems = append(protoItems, &nutritionv1.MealItem{
				Id:                   it.ID,
				MealLogId:            it.MealLogID,
				Name:                 it.Name,
				EstimatedWeightGrams: it.EstimatedWeightGrams,
				Calories:             it.Calories,
				ProteinGrams:         it.ProteinGrams,
				CarbsGrams:           it.CarbsGrams,
				FatGrams:             it.FatGrams,
				FiberGrams:           it.FiberGrams,
				SodiumMg:             it.SodiumMg,
				ConfidenceScore:      it.ConfidenceScore,
				Ingredients:          it.Ingredients,
			})
		}

		protoMeals = append(protoMeals, &nutritionv1.MealLog{
			Id:            m.ID,
			UserUuid:      m.UserUUID,
			MealType:      m.MealType,
			LoggedAt:      m.LoggedAt.Format(time.RFC3339),
			Notes:         m.Notes,
			Items:         protoItems,
			TotalCalories: m.TotalCalories,
			TotalProtein:  m.TotalProtein,
			TotalCarbs:    m.TotalCarbs,
			TotalFat:      m.TotalFat,
		})
	}

	goalsProto := &nutritionv1.NutritionGoals{}
	if summary.Goals != nil {
		goalsProto.UserUuid = summary.Goals.UserUUID
		goalsProto.CaloriesTarget = summary.Goals.CaloriesTarget
		goalsProto.ProteinGrams = summary.Goals.ProteinGrams
		goalsProto.CarbsGrams = summary.Goals.CarbsGrams
		goalsProto.FatGrams = summary.Goals.FatGrams
		goalsProto.FiberGrams = summary.Goals.FiberGrams
		goalsProto.SodiumMg = summary.Goals.SodiumMg
	}

	return &nutritionv1.DailyNutritionSummary{
		Date:          summary.Date,
		TotalCalories: summary.TotalCalories,
		TotalProtein:  summary.TotalProtein,
		TotalCarbs:    summary.TotalCarbs,
		TotalFat:      summary.TotalFat,
		TotalFiber:    summary.TotalFiber,
		TotalSodium:   summary.TotalSodium,
		Goals:         goalsProto,
		Meals:         protoMeals,
	}, nil
}

func (s *NutritionGRPCServer) AddManualMeal(ctx context.Context, req *nutritionv1.AddManualMealRequest) (*nutritionv1.MealLog, error) {
	userUUID := extractUserUUID(ctx, req.GetUserUuid())
	loggedAt := time.Now()
	if req.GetLoggedAt() != "" {
		if t, err := time.Parse(time.RFC3339, req.GetLoggedAt()); err == nil {
			loggedAt = t
		}
	}

	var items []core.MealItem
	for _, it := range req.GetItems() {
		items = append(items, core.MealItem{
			ID:                   uuid.New().String(),
			Name:                 it.GetName(),
			EstimatedWeightGrams: it.GetEstimatedWeightGrams(),
			Calories:             it.GetCalories(),
			ProteinGrams:         it.GetProteinGrams(),
			CarbsGrams:           it.GetCarbsGrams(),
			FatGrams:             it.GetFatGrams(),
			FiberGrams:           it.GetFiberGrams(),
			SodiumMg:             it.GetSodiumMg(),
			ConfidenceScore:      1.0,
			Ingredients:          it.GetIngredients(),
		})
	}

	mealLog := &core.MealLog{
		UserUUID: userUUID,
		MealType: req.GetMealType(),
		LoggedAt: loggedAt,
		Notes:    req.GetNotes(),
		Items:    items,
	}

	saved, err := s.repo.CreateMealLog(ctx, mealLog)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create meal: %v", err)
	}

	var protoItems []*nutritionv1.MealItem
	for _, it := range saved.Items {
		protoItems = append(protoItems, &nutritionv1.MealItem{
			Id:                   it.ID,
			MealLogId:            it.MealLogID,
			Name:                 it.Name,
			EstimatedWeightGrams: it.EstimatedWeightGrams,
			Calories:             it.Calories,
			ProteinGrams:         it.ProteinGrams,
			CarbsGrams:           it.CarbsGrams,
			FatGrams:             it.FatGrams,
			FiberGrams:           it.FiberGrams,
			SodiumMg:             it.SodiumMg,
			ConfidenceScore:      it.ConfidenceScore,
			Ingredients:          it.Ingredients,
		})
	}

	return &nutritionv1.MealLog{
		Id:            saved.ID,
		UserUuid:      saved.UserUUID,
		MealType:      saved.MealType,
		LoggedAt:      saved.LoggedAt.Format(time.RFC3339),
		Notes:         saved.Notes,
		Items:         protoItems,
		TotalCalories: saved.TotalCalories,
		TotalProtein:  saved.TotalProtein,
		TotalCarbs:    saved.TotalCarbs,
		TotalFat:      saved.TotalFat,
	}, nil
}

func (s *NutritionGRPCServer) DeleteMealLog(ctx context.Context, req *nutritionv1.DeleteMealLogRequest) (*nutritionv1.DeleteResponse, error) {
	userUUID := extractUserUUID(ctx, req.GetUserUuid())
	if err := s.repo.DeleteMealLog(ctx, userUUID, req.GetMealLogId()); err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to delete meal log: %v", err)
	}

	return &nutritionv1.DeleteResponse{
		Success: true,
		Message: "meal log deleted successfully",
	}, nil
}

func (s *NutritionGRPCServer) UpdateMealItem(ctx context.Context, req *nutritionv1.UpdateMealItemRequest) (*nutritionv1.MealItem, error) {
	userUUID := extractUserUUID(ctx, req.GetUserUuid())
	item := &core.MealItem{
		ID:                   req.GetItemId(),
		Name:                 req.GetName(),
		EstimatedWeightGrams: req.GetEstimatedWeightGrams(),
		Calories:             req.GetCalories(),
		ProteinGrams:         req.GetProteinGrams(),
		CarbsGrams:           req.GetCarbsGrams(),
		FatGrams:             req.GetFatGrams(),
		FiberGrams:           req.GetFiberGrams(),
	}

	updated, err := s.repo.UpdateMealItem(ctx, userUUID, item)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to update meal item: %v", err)
	}

	return &nutritionv1.MealItem{
		Id:                   updated.ID,
		MealLogId:            updated.MealLogID,
		Name:                 updated.Name,
		EstimatedWeightGrams: updated.EstimatedWeightGrams,
		Calories:             updated.Calories,
		ProteinGrams:         updated.ProteinGrams,
		CarbsGrams:           updated.CarbsGrams,
		FatGrams:             updated.FatGrams,
		FiberGrams:           updated.FiberGrams,
		SodiumMg:             updated.SodiumMg,
		ConfidenceScore:      updated.ConfidenceScore,
		Ingredients:          updated.Ingredients,
	}, nil
}

func (s *NutritionGRPCServer) DeleteMealItem(ctx context.Context, req *nutritionv1.DeleteMealItemRequest) (*nutritionv1.DeleteResponse, error) {
	userUUID := extractUserUUID(ctx, req.GetUserUuid())
	if err := s.repo.DeleteMealItem(ctx, userUUID, req.GetItemId()); err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to delete meal item: %v", err)
	}

	return &nutritionv1.DeleteResponse{
		Success: true,
		Message: "meal item deleted successfully",
	}, nil
}

func (s *NutritionGRPCServer) ProcessImage(ctx context.Context, req *nutritionv1.ProcessImageRequest) (*nutritionv1.JobStatus, error) {
	userUUID := extractUserUUID(ctx, req.GetUserUuid())
	if len(req.GetImageData()) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "image data cannot be empty")
	}

	jobID := uuid.New().String()
	err := s.worker.Enqueue(
		ctx,
		jobID,
		userUUID,
		req.GetImageData(),
		req.GetMimeType(),
		req.GetMealType(),
		req.GetNotes(),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to enqueue job: %v", err)
	}

	return &nutritionv1.JobStatus{
		JobId:     jobID,
		UserUuid:  userUUID,
		Status:    "PENDING",
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	}, nil
}

func (s *NutritionGRPCServer) GetJobStatus(ctx context.Context, req *nutritionv1.GetJobStatusRequest) (*nutritionv1.JobStatus, error) {
	userUUID := extractUserUUID(ctx, req.GetUserUuid())
	job, err := s.repo.GetJob(ctx, userUUID, req.GetJobId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "job not found: %v", err)
	}

	var protoItems []*nutritionv1.MealItem
	for _, it := range job.DetectedItems {
		protoItems = append(protoItems, &nutritionv1.MealItem{
			Id:                   it.ID,
			MealLogId:            it.MealLogID,
			Name:                 it.Name,
			EstimatedWeightGrams: it.EstimatedWeightGrams,
			Calories:             it.Calories,
			ProteinGrams:         it.ProteinGrams,
			CarbsGrams:           it.CarbsGrams,
			FatGrams:             it.FatGrams,
			FiberGrams:           it.FiberGrams,
			SodiumMg:             it.SodiumMg,
			ConfidenceScore:      it.ConfidenceScore,
			Ingredients:          it.Ingredients,
		})
	}

	return &nutritionv1.JobStatus{
		JobId:         job.JobID,
		UserUuid:      job.UserUUID,
		Status:        job.Status,
		ErrorMessage:  job.ErrorMessage,
		MealLogId:     job.MealLogID,
		CreatedAt:     job.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     job.UpdatedAt.Format(time.RFC3339),
		DetectedItems: protoItems,
	}, nil
}

func StartGRPCServer(
	server *NutritionGRPCServer,
	worker core.ImageQueueWorker,
	lc fx.Lifecycle,
	log *zap.Logger,
) (*grpc.Server, error) {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50052"
	}

	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(32*1024*1024),
		grpc.MaxSendMsgSize(32*1024*1024),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	nutritionv1.RegisterNutritionServiceServer(grpcServer, server)

	if lc != nil {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				worker.Start(ctx)

				lis, err := net.Listen("tcp", ":"+port)
				if err != nil {
					return fmt.Errorf("failed to listen on port %s: %w", port, err)
				}
				log.Info("Nutrition gRPC server listening", zap.String("port", port))
				go func() {
					if err := grpcServer.Serve(lis); err != nil {
						log.Error("gRPC server terminated", zap.Error(err))
					}
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				worker.Stop()
				grpcServer.GracefulStop()
				log.Info("Nutrition gRPC server stopped")
				return nil
			},
		})
	}

	return grpcServer, nil
}
