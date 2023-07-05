CREATE TABLE jobs
(
    id               SERIAL PRIMARY KEY,
    jobId            INT UNIQUE,
    serviceId        INT,
    userId           TEXT,
    jobStartTypeId   INT,
    jobStartDate TIMESTAMPTZ,
    jobStartFromTime TIMETZ,
    jobState         INT,
    jobCity          INT,
    jobDetails       TEXT,
    jobQuoteTimeLast BIGINT,
    jobStatusId      INT,
    createDate       BIGINT
);
