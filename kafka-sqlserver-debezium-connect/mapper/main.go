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
	batchSize          int
}

type services struct {
	kafkaReader *kafka.Reader
	mapper      Mapper
	repository  Repository
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
	Map(event []byte, partition int) (JobMasterRow, error)
}

type Repository interface {
	Upsert(jobs []JobMasterRow) error
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
	var currentBatch []JobMasterRow
	for {
		m, err := svc.kafkaReader.FetchMessage(ctx)
		if err != nil {
			break
		}

		logrus.Tracef("message %v at topic/partition/offset %v/%v/%v: %s = %s\n", m.Value, m.Topic, m.Partition, m.Offset, string(m.Key), string(m.Value))

		rowEvent, err := svc.mapper.Map(m.Value, m.Partition)
		if err != nil {
			log.Fatal("failed to map message:", err)
		}

		currentBatch = append(currentBatch, rowEvent)

		if len(currentBatch) < c.batchSize {
			continue
		}

		err = svc.repository.Upsert(currentBatch)
		if err != nil {
			log.Fatal("failed to upsert message:", err)
		}

		if err := svc.kafkaReader.CommitMessages(ctx, m); err != nil {
			log.Fatal("failed to commit messages:", err)
		}

		currentBatch = []JobMasterRow{}
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
			return config{}, fmt.Errorf("KAFKA_MAX_BYTES is not a valid int: %w", err)
		}
		maxBytes = atoi
	}

	postgresDataSource := os.Getenv("POSTGRES_DATA_SOURCE")
	if postgresDataSource == "" {
		return config{}, fmt.Errorf("POSTGRES_DATA_SOURCE is not set")
	}

	batchSize := 1000
	bs := os.Getenv("BATCH_SIZE")
	if bs != "" {
		atoi, err := strconv.Atoi(bs)
		if err != nil {
			return config{}, fmt.Errorf("BATCH_SIZE is not a valid int: %w", err)
		}
		batchSize = atoi
	}

	return config{
		brokers:            strings.Split(kb, ","),
		consumerGroupID:    cgid,
		topic:              topic,
		maxBytes:           maxBytes,
		postgresDataSource: postgresDataSource,
		batchSize:          batchSize,
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
