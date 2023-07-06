package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	_ "github.com/lib/pq"
)

type config struct {
	brokers            []string
	consumerGroupID    string
	topic              string
	maxBytes           int
	postgresDataSource string
}

type services struct {
	kafkaReader *kafka.Reader
	mapper      Mapper
	repository  Repository
}

type JobMasterRow struct {
	JobId            int64     `json:"JobId" db:"jobId"`
	ServiceId        int64     `json:"ServiceId" db:"serviceId"`
	UserId           string    `json:"UserId" db:"userId"`
	JobStartTypeId   int64     `json:"JobStartTypeId" db:"jobStartTypeId"`
	JobStartDate     time.Time `json:"JobStartDate" db:"jobStartDate"`
	JobStartFromTime int64     `json:"JobStartFromTime" db:"jobStartFromTime"`
	JobState         int64     `json:"JobState" db:"jobState"`
	JobCity          int64     `json:"JobCity" db:"jobCity"`
	JobDetails       string    `json:"JobDetails" db:"jobDetails"`
	JobQuoteTimeLast time.Time `json:"JobQuoteTimeLast" db:"jobQuoteTimeLast"`
	JobStatusId      int64     `json:"JobStatusId" db:"jobStatusId"`
	CreateDate       time.Time `json:"CreateDate" db:"createDate"`
}

type Mapper interface {
	Map(event []byte) (JobMasterRow, error)
}

type Repository interface {
	Upsert(job JobMasterRow) error
}

func main() {
	c, err := initConfigs()
	if err != nil {
		logrus.Fatal(err)
	}

	svc, deferF, err := initServices(c)
	if err != nil {
		logrus.Fatal(err)
	}
	defer deferF()

	ctx := context.Background()
	for {
		m, err := svc.kafkaReader.FetchMessage(ctx)
		if err != nil {
			break
		}

		logrus.Tracef("message %v at topic/partition/offset %v/%v/%v: %s = %s\n", m.Value, m.Topic, m.Partition, m.Offset, string(m.Key), string(m.Value))

		rowEvent, err := svc.mapper.Map(m.Value)
		if err != nil {
			log.Fatal("failed to map message:", err)
		}

		err = svc.repository.Upsert(rowEvent)
		if err != nil {
			log.Fatal("failed to upsert message:", err)
		}

		if err := svc.kafkaReader.CommitMessages(ctx, m); err != nil {
			log.Fatal("failed to commit messages:", err)
		}
	}
}

func initConfigs() (config, error) {
	kb := os.Getenv("KAFKA_BROKERS")
	if kb == "" {
		return config{}, fmt.Errorf("KAFKA_BROKERS is not set")
	}

	cgid := os.Getenv("KAFKA_CONSUMER_GROUP_ID")
	if cgid == "" {
		return config{}, fmt.Errorf("KAFKA_CONSUMER_GROUP_ID is not set")
	}

	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		return config{}, fmt.Errorf("KAFKA_TOPIC is not set")
	}

	var maxBytes int = 10e6
	mb := os.Getenv("KAFKA_MAX_BYTES")
	if mb != "" {
		atoi, err := strconv.Atoi(mb)
		if err != nil {
			return config{}, fmt.Errorf("KAFKA_TOPIC is not set: %w", err)
		}
		maxBytes = atoi
	}

	postgresDataSource := os.Getenv("POSTGRES_DATA_SOURCE")
	if postgresDataSource == "" {
		return config{}, fmt.Errorf("POSTGRES_DATA_SOURCE is not set")
	}

	return config{
		brokers:            strings.Split(kb, ","),
		consumerGroupID:    cgid,
		topic:              topic,
		maxBytes:           maxBytes,
		postgresDataSource: postgresDataSource,
	}, nil

}

func initServices(c config) (services, func(), error) {
	db, err := sqlx.Open("postgres", c.postgresDataSource)
	if err != nil {
		return services{}, nil, fmt.Errorf("failed to connect to db: %w", err)
	}

	kr := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  c.brokers,
		GroupID:  c.consumerGroupID,
		Topic:    c.topic,
		MaxBytes: c.maxBytes,
	})

	return services{
			mapper: &JSONMapper{},
			repository: &PostgresRepository{
				db: db,
			},
			kafkaReader: kr,
		}, func() {
			err := db.Close()
			if err != nil {
				logrus.Warning("failed to close db connection: %w", err)
			}
		}, nil
}
