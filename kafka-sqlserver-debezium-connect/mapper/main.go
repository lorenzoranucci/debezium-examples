package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/caarlos0/env/v9"
	"github.com/jmoiron/sqlx"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	_ "github.com/lib/pq"
)

var svc = services{}
var cnf = config{}
var deferF func()

func main() {
	defer deferF()

	currentBatch := make([]kafka.Message, 0, cnf.BatchSize)
	for {
		m, err := fetchMessageWithDeadline()

		isDeadlineExceeded := errors.Is(err, context.DeadlineExceeded)
		if err != nil && !isDeadlineExceeded {
			logrus.Fatal(err)
		}

		if !isDeadlineExceeded {
			logrus.Tracef("message %s at topic/partition/offset %v/%v/%v: %s = %s\n", m.Value, m.Topic, m.Partition, m.Offset, string(m.Key), string(m.Value))
			currentBatch = append(currentBatch, m)
		}

		shouldProcessBatch := len(currentBatch) == cnf.BatchSize || isDeadlineExceeded && len(currentBatch) > 0
		if !shouldProcessBatch {
			logrus.
				WithField("currentBatchSize", len(currentBatch)).
				WithField("maxBatchLength", cnf.BatchSize).
				WithField("isDeadlineExceeded", isDeadlineExceeded).
				Debug("skipped message")
			continue
		}

		jobMasterRows, err := svc.mapper.MapBatch(currentBatch)
		if err != nil {
			log.Fatal("failed to map messages:", err)
		}

		err = svc.repository.Upsert(jobMasterRows)
		if err != nil {
			log.Fatal("failed to upsert messages:", err)
		}

		if err := svc.kafkaReader.CommitMessages(context.Background(), currentBatch[len(currentBatch)-1]); err != nil {
			log.Fatal("failed to commit messages:", err)
		}

		currentBatch = []kafka.Message{}
	}
}

func fetchMessageWithDeadline() (kafka.Message, error) {
	ctx, cf := context.WithDeadline(context.Background(), time.Now().Add(cnf.MessageFetchDeadlineInSeconds))
	defer cf()
	return svc.kafkaReader.FetchMessage(ctx)
}

type JobMasterRow struct {
	JobId               int64     `json:"JobId"`
	ServiceId           int64     `json:"ServiceId"`
	UserId              string    `json:"UserId"`
	JobStartTypeId      int64     `json:"JobStartTypeId"`
	JobStartDate        time.Time `json:"JobStartDate"`
	JobStartFromTime    int64     `json:"JobStartFromTime"`
	JobState            int64     `json:"JobState"`
	JobCity             int64     `json:"JobCity"`
	JobDetails          string    `json:"JobDetails"`
	JobQuoteTimeLast    time.Time `json:"JobQuoteTimeLast"`
	JobStatusId         int64     `json:"JobStatusId"`
	CreateDate          time.Time `json:"CreateDate"`
	_FromKafkaPartition int
}

type Mapper interface {
	MapBatch([]kafka.Message) ([]JobMasterRow, error)
	Map(kafka.Message) (JobMasterRow, error)
}

type Repository interface {
	Upsert(jobs []JobMasterRow) error
}

type config struct {
	Brokers                       []string      `env:"KAFKA_BROKERS" envSeparator:","`
	ConsumerGroupID               string        `env:"KAFKA_CONSUMER_GROUP_ID"`
	Topic                         string        `env:"KAFKA_TOPIC"`
	MaxBytes                      int           `env:"KAFKA_MAX_BYTES" envDefault:"10000000"`
	PostgresDataSource            string        `env:"POSTGRES_DATA_SOURCE"`
	BatchSize                     int           `env:"BATCH_SIZE" envDefault:"1000"`
	MessageFetchDeadlineInSeconds time.Duration `env:"MESSAGE_FETCH_DEADLINE_IN_SECONDS" envDefault:"3s"`
	LogLevel                      logrus.Level  `env:"LOG_LEVEL" envDefault:"info"`
}

type services struct {
	kafkaReader *kafka.Reader
	mapper      Mapper
	repository  Repository
}

func init() {
	var err error
	cnf, err = initConfigs()
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.SetLevel(cnf.LogLevel)

	svc, deferF, err = initServices()
	if err != nil {
		logrus.Fatal(err)
	}
}

func initConfigs() (config, error) {
	cnf := config{}
	err := env.Parse(&cnf)
	if err != nil {
		return config{}, err
	}

	return cnf, nil
}

func initServices() (services, func(), error) {
	db, err := sqlx.Open("postgres", cnf.PostgresDataSource)
	if err != nil {
		return services{}, nil, fmt.Errorf("failed to connect to db: %w", err)
	}

	kr := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cnf.Brokers,
		GroupID:  cnf.ConsumerGroupID,
		Topic:    cnf.Topic,
		MaxBytes: cnf.MaxBytes,
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
