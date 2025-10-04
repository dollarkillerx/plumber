package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/plumber/plumber/pkg/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// Storage 存储接口
type Storage interface {
	// Agent相关
	CreateAgent(ctx context.Context, agent *models.Agent) error
	GetAgent(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	ListAgents(ctx context.Context) ([]*models.Agent, error)
	UpdateAgent(ctx context.Context, agent *models.Agent) error
	UpdateAgentHeartbeat(ctx context.Context, id uuid.UUID) error
	UpdateAgentStatus(ctx context.Context, id uuid.UUID, status string) error
	DeleteAgent(ctx context.Context, id uuid.UUID) error

	// Task相关
	CreateTask(ctx context.Context, task *models.Task) error
	GetTask(ctx context.Context, id uuid.UUID) (*models.Task, error)
	ListTasks(ctx context.Context) ([]*models.Task, error)
	UpdateTask(ctx context.Context, task *models.Task) error
	DeleteTask(ctx context.Context, id uuid.UUID) error

	// TaskExecution相关
	CreateExecution(ctx context.Context, execution *models.TaskExecution) error
	GetExecution(ctx context.Context, id uuid.UUID) (*models.TaskExecution, error)
	ListExecutionsByTaskID(ctx context.Context, taskID uuid.UUID) ([]*models.TaskExecution, error)
	UpdateExecution(ctx context.Context, execution *models.TaskExecution) error

	// StepExecution相关
	CreateStepExecution(ctx context.Context, step *models.StepExecution) error
	UpdateStepExecution(ctx context.Context, step *models.StepExecution) error
	GetStepExecution(ctx context.Context, id uuid.UUID) (*models.StepExecution, error)
	GetPendingStepsForAgent(ctx context.Context, agentID uuid.UUID, limit int) ([]*models.StepExecution, error)
	MarkStepAsAssigned(ctx context.Context, stepID uuid.UUID) error

	// User相关
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUserByToken(ctx context.Context, token string) (*models.User, error)
	UpdateUserToken(ctx context.Context, userID uuid.UUID, token string) error

	Close() error
}

// PostgresStorage PostgreSQL存储实现
type PostgresStorage struct {
	db *gorm.DB
}

// NewPostgresStorage 创建PostgreSQL存储
func NewPostgresStorage(dsn string, debug bool) (*PostgresStorage, error) {
	logLevel := logger.Silent
	if debug {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(
		&models.Agent{},
		&models.Task{},
		&models.TaskExecution{},
		&models.StepExecution{},
		&models.User{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &PostgresStorage{db: db}, nil
}

// Agent相关方法
func (s *PostgresStorage) CreateAgent(ctx context.Context, agent *models.Agent) error {
	return s.db.WithContext(ctx).Create(agent).Error
}

func (s *PostgresStorage) GetAgent(ctx context.Context, id uuid.UUID) (*models.Agent, error) {
	var agent models.Agent
	if err := s.db.WithContext(ctx).First(&agent, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &agent, nil
}

func (s *PostgresStorage) ListAgents(ctx context.Context) ([]*models.Agent, error) {
	var agents []*models.Agent
	if err := s.db.WithContext(ctx).Find(&agents).Error; err != nil {
		return nil, err
	}
	return agents, nil
}

func (s *PostgresStorage) UpdateAgent(ctx context.Context, agent *models.Agent) error {
	return s.db.WithContext(ctx).Save(agent).Error
}

func (s *PostgresStorage) UpdateAgentHeartbeat(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Model(&models.Agent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_heartbeat": time.Now(),
			"status":         "online",
		}).Error
}

func (s *PostgresStorage) UpdateAgentStatus(ctx context.Context, id uuid.UUID, status string) error {
	return s.db.WithContext(ctx).Model(&models.Agent{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (s *PostgresStorage) DeleteAgent(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Delete(&models.Agent{}, "id = ?", id).Error
}

// Task相关方法
func (s *PostgresStorage) CreateTask(ctx context.Context, task *models.Task) error {
	return s.db.WithContext(ctx).Create(task).Error
}

func (s *PostgresStorage) GetTask(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	var task models.Task
	if err := s.db.WithContext(ctx).First(&task, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *PostgresStorage) ListTasks(ctx context.Context) ([]*models.Task, error) {
	var tasks []*models.Task
	if err := s.db.WithContext(ctx).Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *PostgresStorage) UpdateTask(ctx context.Context, task *models.Task) error {
	return s.db.WithContext(ctx).Save(task).Error
}

func (s *PostgresStorage) DeleteTask(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Delete(&models.Task{}, "id = ?", id).Error
}

// TaskExecution相关方法
func (s *PostgresStorage) CreateExecution(ctx context.Context, execution *models.TaskExecution) error {
	return s.db.WithContext(ctx).Create(execution).Error
}

func (s *PostgresStorage) GetExecution(ctx context.Context, id uuid.UUID) (*models.TaskExecution, error) {
	var execution models.TaskExecution
	if err := s.db.WithContext(ctx).
		Preload("Steps").
		First(&execution, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &execution, nil
}

func (s *PostgresStorage) ListExecutionsByTaskID(ctx context.Context, taskID uuid.UUID) ([]*models.TaskExecution, error) {
	var executions []*models.TaskExecution
	if err := s.db.WithContext(ctx).
		Preload("Steps", func(db *gorm.DB) *gorm.DB {
			return db.Order("step_index ASC")
		}).
		Where("task_id = ?", taskID).
		Order("created_at DESC").
		Limit(20).
		Find(&executions).Error; err != nil {
		return nil, err
	}
	return executions, nil
}

func (s *PostgresStorage) UpdateExecution(ctx context.Context, execution *models.TaskExecution) error {
	return s.db.WithContext(ctx).Save(execution).Error
}

// StepExecution相关方法
func (s *PostgresStorage) CreateStepExecution(ctx context.Context, step *models.StepExecution) error {
	return s.db.WithContext(ctx).Create(step).Error
}

func (s *PostgresStorage) UpdateStepExecution(ctx context.Context, step *models.StepExecution) error {
	return s.db.WithContext(ctx).Save(step).Error
}

func (s *PostgresStorage) GetStepExecution(ctx context.Context, id uuid.UUID) (*models.StepExecution, error) {
	var step models.StepExecution
	if err := s.db.WithContext(ctx).First(&step, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &step, nil
}

func (s *PostgresStorage) GetPendingStepsForAgent(ctx context.Context, agentID uuid.UUID, limit int) ([]*models.StepExecution, error) {
	var steps []*models.StepExecution

	// 使用事务 + FOR UPDATE 行锁，防止并发问题
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 查询待执行的步骤
		var candidates []*models.StepExecution
		if err := tx.
			Where("agent_id = ? AND status = ? AND assigned = ?", agentID, "pending", false).
			Order("created_at ASC").
			Limit(limit).
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Find(&candidates).Error; err != nil {
			return err
		}

		// 对每个候选步骤，检查是否有未完成的前序步骤
		for _, candidate := range candidates {
			// 检查同一个 execution 中是否有更早的步骤还在运行
			var runningPrevSteps int64
			if err := tx.Model(&models.StepExecution{}).
				Where("execution_id = ? AND step_index < ? AND status IN ?",
					candidate.ExecutionID, candidate.StepIndex, []string{"pending", "running"}).
				Count(&runningPrevSteps).Error; err != nil {
				return err
			}

			// 如果有前序步骤未完成，跳过这个步骤
			if runningPrevSteps > 0 {
				continue
			}

			// 这个步骤可以执行
			steps = append(steps, candidate)
		}

		// 在同一事务中立即标记为已分配
		if len(steps) > 0 {
			stepIDs := make([]uuid.UUID, len(steps))
			for i, step := range steps {
				stepIDs[i] = step.ID
			}
			if err := tx.Model(&models.StepExecution{}).
				Where("id IN ?", stepIDs).
				Update("assigned", true).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return steps, nil
}

func (s *PostgresStorage) MarkStepAsAssigned(ctx context.Context, stepID uuid.UUID) error {
	// 这个方法现在不再需要，因为在 GetPendingStepsForAgent 中已经标记了
	// 但保留以防其他地方使用
	return s.db.WithContext(ctx).
		Model(&models.StepExecution{}).
		Where("id = ? AND assigned = ?", stepID, false).
		Update("assigned", true).Error
}

// User相关方法
func (s *PostgresStorage) CreateUser(ctx context.Context, user *models.User) error {
	return s.db.WithContext(ctx).Create(user).Error
}

func (s *PostgresStorage) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, "username = ?", username).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *PostgresStorage) GetUserByToken(ctx context.Context, token string) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, "token = ?", token).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *PostgresStorage) UpdateUserToken(ctx context.Context, userID uuid.UUID, token string) error {
	return s.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Update("token", token).Error
}

func (s *PostgresStorage) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
