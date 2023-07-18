package main

import (
	"context"
	"errors"
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
	brokers                       []string
	consumerGroupID               string
	topic                         string
	maxBytes                      int
	postgresDataSource            string
	batchSize                     int
	messageFetchDeadlineInSeconds time.Duration
	logLevel                      logrus.Level
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
	MapBatch([]kafka.Message) ([]JobMasterRow, error)
	Map(kafka.Message) (JobMasterRow, error)
}

type Repository interface {
	Upsert(jobs []JobMasterRow) error
}

var svc = services{}
var cnf = config{}
var deferF func()

func main() {
	defer deferF()

	currentBatch := make([]kafka.Message, 0, cnf.batchSize)
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

		shouldProcessBatch := len(currentBatch) == cnf.batchSize || isDeadlineExceeded && len(currentBatch) > 0
		if !shouldProcessBatch {
			logrus.
				WithField("currentBatchSize", len(currentBatch)).
				WithField("maxBatchLength", cnf.batchSize).
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
	ctx, cf := context.WithDeadline(context.Background(), time.Now().Add(cnf.messageFetchDeadlineInSeconds))
	defer cf()
	return svc.kafkaReader.FetchMessage(ctx)
}

func init() {
	var err error
	cnf, err = initConfigs()
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.SetLevel(cnf.logLevel)

	svc, deferF, err = initServices(cnf)
	if err != nil {
		logrus.Fatal(err)
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

	messageFetchDeadlineIn := 2 * time.Second
	d := os.Getenv("MESSAGE_FETCH_DEADLINE_IN_SECONDS")
	if d != "" {
		atoi, err := strconv.Atoi(d)
		if err != nil {
			return config{}, fmt.Errorf("MESSAGE_FETCH_DEADLINE_IN_SECONDS is not a valid int: %w", err)
		}
		messageFetchDeadlineIn = time.Second * time.Duration(atoi)
	}

	logLevel := logrus.TraceLevel
	ll := os.Getenv("LOG_LEVEL")
	if ll != "" {
		var err error
		logLevel, err = logrus.ParseLevel(ll)
		if err != nil {
			return config{}, fmt.Errorf("LOG_LEVEL is not a valid logrus level: %w", err)
		}
	}

	return config{
		brokers:                       strings.Split(kb, ","),
		consumerGroupID:               cgid,
		topic:                         topic,
		maxBytes:                      maxBytes,
		postgresDataSource:            postgresDataSource,
		batchSize:                     batchSize,
		messageFetchDeadlineInSeconds: messageFetchDeadlineIn,
		logLevel:                      logLevel,
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
