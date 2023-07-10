CREATE TABLE jobs
(
    id                  SERIAL PRIMARY KEY,
    jobId               INT UNIQUE,
    serviceId           BIGINT,
    userId              VARCHAR(50),
    jobStartTypeId      INT,
    jobStartDate        TIMESTAMP,
    jobStartFromTime    INT,
    jobState            INT,
    jobCity             INT,
    jobDetails          VARCHAR(2000),
    jobQuoteTimeLast    TIMESTAMP,
    jobStatusId         INT,
    createDate          TIMESTAMP,
    _lastUpdatedAt      TIMESTAMP,
    _fromKafkaPartition TEXT
);

CREATE INDEX jobs__last_updated_at_index ON jobs (_lastUpdatedAt);
