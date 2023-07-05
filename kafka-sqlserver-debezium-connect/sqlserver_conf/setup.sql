create database inventory
go

use inventory

-- Create the Users table
CREATE TABLE dbo.Users (
                       ID INT IDENTITY(1,1) PRIMARY KEY,
                       FirstName NVARCHAR(50),
                       LastName NVARCHAR(50),
                       Email NVARCHAR(100)
);

-- Insert some sample records
INSERT INTO dbo.Users (FirstName, LastName, Email) VALUES ('John', 'Doe', 'john.doe@example.com');
INSERT INTO dbo.Users (FirstName, LastName, Email) VALUES ('Jane', 'Smith', 'jane.smith@example.com');
INSERT INTO dbo.Users (FirstName, LastName, Email) VALUES ('Mike', 'Johnson', 'mike.johnson@example.com');

GO
EXEC sys.sp_cdc_enable_db
GO

SELECT name AS [Filegroup]
FROM sys.filegroups;

EXEC sys.sp_cdc_enable_table
     @source_schema = N'dbo',
     @source_name   = N'Users',
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

