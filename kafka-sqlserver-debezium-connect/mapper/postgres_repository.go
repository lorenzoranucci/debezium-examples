package main

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type PostgresRepository struct {
	db *sqlx.DB
}

func (p *PostgresRepository) Upsert(job JobMasterRow) error {
	query := `INSERT INTO jobs (
                  jobId,
                  serviceId,
                  userId,
                  jobStartTypeId,
                  jobStartDate,
                  jobStartFromTime,
                  jobState,
                  jobCity,
                  jobDetails,
                  jobQuoteTimeLast,
                  jobStatusId,
                  createDate)
VALUES (:jobId,
        :serviceId,
        :userId,
        :jobStartTypeId,
        :jobStartDate,
        :jobStartFromTime,
        :jobState,
        :jobCity,
        :jobDetails,
        :jobQuoteTimeLast,
        :jobStatusId,
        :createDate)
ON CONFLICT (jobId) DO UPDATE
    SET jobId            = :jobId,
        serviceId        = :serviceId,
        userId           = :userId,
        jobStartTypeId   = :jobStartTypeId,
        jobStartDate     = :jobStartDate,
        jobStartFromTime = :jobStartFromTime,
        jobState         = :jobState,
        jobCity          = :jobCity,
        jobDetails       = :jobDetails,
        jobQuoteTimeLast = :jobQuoteTimeLast,
        jobStatusId      = :jobStatusId,
        createDate       = :createDate;`

	_, err := p.db.NamedExec(query, job)
	if err != nil {
		return fmt.Errorf("failed to upsert job: %w", err)
	}

	return nil
}
