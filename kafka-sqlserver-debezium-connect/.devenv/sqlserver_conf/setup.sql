create database yerelservis
go

use yerelservis

create table dbo.tblJobMaster
(
    JobId                      bigint                  not null
        constraint PK_tbljobmaster_1
            primary key
                with (fillfactor = 90),
    ServiceId                  bigint                  not null,
    UserId                     nvarchar(50)            not null,
    JobStartTypeId             int,
    JobStartDate               datetime,
    JobStartFromTime           int,
    JobState                   int,
    JobCity                    int,
    JobDetails                 nvarchar(2000),
    JobQuoteTimeLast           datetime                not null,
    JobStatusId                int,
    CreateDate                 datetime                not null
);
go

ALTER DATABASE yerelservis
    SET ALLOW_SNAPSHOT_ISOLATION ON;
go

INSERT INTO dbo.tblJobMaster (JobId, ServiceId, UserId, JobStartTypeId, JobStartDate, JobStartFromTime, JobState, JobCity, JobDetails, JobQuoteTimeLast, JobStatusId, CreateDate)
VALUES (1, 1001, 'User1', 1, '2023-07-01', 800, 1, 1, 'Job Details 1', '2023-07-01 09:00:00', 1, '2023-07-01 09:15:00');

INSERT INTO dbo.tblJobMaster (JobId, ServiceId, UserId, JobStartTypeId, JobStartDate, JobStartFromTime, JobState, JobCity, JobDetails, JobQuoteTimeLast, JobStatusId, CreateDate)
VALUES (2, 1002, 'User2', 2, '2023-07-02', 900, 2, 2, 'Job Details 2', '2023-07-02 10:30:00', 2, '2023-07-02 10:45:00');

INSERT INTO dbo.tblJobMaster (JobId, ServiceId, UserId, JobStartTypeId, JobStartDate, JobStartFromTime, JobState, JobCity, JobDetails, JobQuoteTimeLast, JobStatusId, CreateDate)
VALUES (3, 1003, 'User3', 1, '2023-07-03', 1100, 1, 3, 'Job Details 3', '2023-07-03 12:00:00', 1, '2023-07-03 12:15:00');

INSERT INTO dbo.tblJobMaster (JobId, ServiceId, UserId, JobStartTypeId, JobStartDate, JobStartFromTime, JobState, JobCity, JobDetails, JobQuoteTimeLast, JobStatusId, CreateDate)
VALUES (4, 1004, 'User4', 2, '2023-07-04', 1300, 2, 4, 'Job Details 4', '2023-07-04 14:30:00', 2, '2023-07-04 14:45:00');

GO
EXEC sys.sp_cdc_enable_db
GO

SELECT name AS [Filegroup]
FROM sys.filegroups;

EXEC sys.sp_cdc_enable_table
     @source_schema = N'dbo',
     @source_name   = N'tblJobMaster',
     @role_name     = null,
     @filegroup_name = N'PRIMARY',
     @supports_net_changes = 0
GO

CREATE LOGIN [user1] WITH PASSWORD = 'Rootroot1.';
CREATE USER [user1] FOR LOGIN [user1];
GO

EXEC sp_addrolemember 'db_owner', 'user1';
GO

GRANT EXECUTE, SELECT, VIEW CHANGE TRACKING ON SCHEMA::dbo TO [user1];
GO

EXEC sys.sp_cdc_help_change_data_capture
GO

USE master;
GRANT VIEW SERVER STATE TO user1;
go
