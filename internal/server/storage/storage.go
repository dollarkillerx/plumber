package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/plumber/plumber/pkg/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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
	UpdateExecution(ctx context.Context, execution *models.TaskExecution) error

	// StepExecution相关
	CreateStepExecution(ctx context.Context, step *models.StepExecution) error
	UpdateStepExecution(ctx context.Context, step *models.StepExecution) error
	GetStepExecution(ctx context.Context, id uuid.UUID) (*models.StepExecution, error)

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
