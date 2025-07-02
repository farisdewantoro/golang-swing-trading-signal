package jobs

import (
	"context"
	"fmt"
	"golang-swing-trading-signal/internal/config"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/repository"
	"golang-swing-trading-signal/internal/utils"

	"github.com/sirupsen/logrus"
)

type JobService interface {
	Get(ctx context.Context, param *models.GetJobParam, opts ...utils.DBOption) ([]models.JobEntity, error)
	RunJobTask(ctx context.Context, jobID uint, opts ...utils.DBOption) error
}

type jobService struct {
	cfg            *config.Config
	log            *logrus.Logger
	jobsRepository repository.JobsRepository
}

func NewJobService(cfg *config.Config, log *logrus.Logger, jobsRepository repository.JobsRepository) JobService {
	return &jobService{
		cfg:            cfg,
		log:            log,
		jobsRepository: jobsRepository,
	}
}

func (s *jobService) Get(ctx context.Context, param *models.GetJobParam, opts ...utils.DBOption) ([]models.JobEntity, error) {
	jobs, err := s.jobsRepository.Get(ctx, param, opts...)
	if err != nil {
		s.log.Error("failed to get jobs", logrus.Fields{
			"error": err,
		})
		return nil, fmt.Errorf("failed to get jobs: %w", err)
	}
	return jobs, nil
}

func (s *jobService) RunJobTask(ctx context.Context, jobID uint, opts ...utils.DBOption) error {
	return s.jobsRepository.RunJobTask(ctx, jobID, opts...)
}
