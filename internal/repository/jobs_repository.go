package repository

import (
	"context"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/utils"

	"gorm.io/gorm"
)

type JobsRepository interface {
	Get(ctx context.Context, param *models.GetJobParam, opts ...utils.DBOption) ([]models.JobEntity, error)
	RunJobTask(ctx context.Context, jobID uint, opts ...utils.DBOption) error
}

type jobsRepository struct {
	db *gorm.DB
}

func NewJobsRepository(db *gorm.DB) JobsRepository {
	return &jobsRepository{db: db}
}

func (r *jobsRepository) Get(ctx context.Context, param *models.GetJobParam, opts ...utils.DBOption) ([]models.JobEntity, error) {
	var jobs []models.JobEntity
	db := utils.ApplyOptions(r.db.WithContext(ctx), opts...)
	db = db.Model(&models.JobEntity{}).Joins("LEFT JOIN task_schedules ON task_schedules.job_id = jobs.id")
	if param.IsActive != nil {
		db = db.Where("task_schedules.is_active = ?", *param.IsActive)
	}
	if len(param.IDs) > 0 {
		db = db.Where("jobs.id IN ?", param.IDs)
	}
	if param.Limit != nil {
		db = db.Limit(*param.Limit)
	}
	if param.WithTaskHistory != nil {
		db = db.Preload("Histories", func(db *gorm.DB) *gorm.DB {
			db = db.Order("created_at DESC")
			if param.WithTaskHistory.Limit != nil {
				db = db.Limit(*param.WithTaskHistory.Limit)
			}
			return db
		})
	}
	result := db.Preload("Schedules").Find(&jobs)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return jobs, nil
}

func (r *jobsRepository) RunJobTask(ctx context.Context, jobID uint, opts ...utils.DBOption) error {
	db := utils.ApplyOptions(r.db.WithContext(ctx), opts...)
	db = db.Model(&models.TaskScheduleEntity{}).Where("job_id = ?", jobID).Update("next_execution", utils.TimeNowWIB())
	return db.Error
}
