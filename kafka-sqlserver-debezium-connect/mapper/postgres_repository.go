package main

import (
	"fmt"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type PostgresRepository struct {
	db *sqlx.DB
}

func (p *PostgresRepository) Upsert(jobs []JobMasterRow) error {
	if len(jobs) == 0 {
		return nil
	}

	// deduplication is needed because the ON CONFLICT clause does not support multiple rows with the same key
	uniqueJobs := deduplicateJobsPickingTheLastOneOnDuplicate(jobs)

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
                  createDate,
                  _fromKafkaPartition,
                  _lastUpdatedAt)
VALUES `
	params := make(map[string]interface{})

	for i, job := range uniqueJobs {
		query += `(:jobId` + strconv.Itoa(i) +
			`, :serviceId` + strconv.Itoa(i) +
			`, :userId` + strconv.Itoa(i) +
			`, :jobStartTypeId` + strconv.Itoa(i) +
			`, :jobStartDate` + strconv.Itoa(i) +
			`, :jobStartFromTime` + strconv.Itoa(i) +
			`, :jobState` + strconv.Itoa(i) +
			`, :jobCity` + strconv.Itoa(i) +
			`, :jobDetails` + strconv.Itoa(i) +
			`, :jobQuoteTimeLast` + strconv.Itoa(i) +
			`, :jobStatusId` + strconv.Itoa(i) +
			`, :createDate` + strconv.Itoa(i) +
			`, :_fromKafkaPartition` + strconv.Itoa(i) +
			`, NOW()), `

		params["jobId"+strconv.Itoa(i)] = job.JobId
		params["serviceId"+strconv.Itoa(i)] = job.ServiceId
		params["userId"+strconv.Itoa(i)] = job.UserId
		params["jobStartTypeId"+strconv.Itoa(i)] = job.JobStartTypeId
		params["jobStartDate"+strconv.Itoa(i)] = job.JobStartDate
		params["jobStartFromTime"+strconv.Itoa(i)] = job.JobStartFromTime
		params["jobState"+strconv.Itoa(i)] = job.JobState
		params["jobCity"+strconv.Itoa(i)] = job.JobCity
		params["jobDetails"+strconv.Itoa(i)] = job.JobDetails
		params["jobQuoteTimeLast"+strconv.Itoa(i)] = job.JobQuoteTimeLast
		params["jobStatusId"+strconv.Itoa(i)] = job.JobStatusId
		params["createDate"+strconv.Itoa(i)] = job.CreateDate
		params["_fromKafkaPartition"+strconv.Itoa(i)] = fmt.Sprintf("%d", job._FromKafkaPartition)
	}

	// remove last comma
	query = query[:len(query)-2]

	query += ` ON CONFLICT (jobId) DO UPDATE
    SET jobId               = excluded.jobId,
        serviceId           = excluded.serviceId,
        userId              = excluded.userId,
        jobStartTypeId      = excluded.jobStartTypeId,
        jobStartDate        = excluded.jobStartDate,
        jobStartFromTime    = excluded.jobStartFromTime,
        jobState            = excluded.jobState,
        jobCity             = excluded.jobCity,
        jobDetails          = excluded.jobDetails,
        jobQuoteTimeLast    = excluded.jobQuoteTimeLast,
        jobStatusId         = excluded.jobStatusId,
        createDate          = excluded.createDate,
        _lastUpdatedAt      = NOW(),
        _fromKafkaPartition = jobs._fromKafkaPartition || ', ' ||excluded._fromKafkaPartition;`

	_, err := p.db.NamedExec(query, params)
	if err != nil {
		return fmt.Errorf("failed to upsert jobs: %w", err)
	}

	return nil
}

func deduplicateJobsPickingTheLastOneOnDuplicate(input []JobMasterRow) []JobMasterRow {
	uniqueJobs := make(map[int64]JobMasterRow)

	for _, job := range input {
		_, ok := uniqueJobs[job.JobId]
		if ok {
			logrus.WithField("job", job).
				Trace("duplicate job found, picking the last one on duplicate")
		}
		uniqueJobs[job.JobId] = job
	}

	uniqueJobSlice := make([]JobMasterRow, 0, len(uniqueJobs))
	for _, job := range uniqueJobs {
		uniqueJobSlice = append(uniqueJobSlice, job)
	}

	return uniqueJobSlice
}
