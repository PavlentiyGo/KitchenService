package core_config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	LoggerCfg *LoggerConfig
	WorkerCfg *WorkerConfig
	ServerCfg *ServerConfig
	KafkaCfg  *KafkaConfig
	DbConfig  *DatabaseConfig
}
type LoggerConfig struct {
	Level  string `envconfig:"LOGGER_LEVEL"    default:"DEBUG"`
	Folder string `envconfig:"LOGGER_FOLDER"` // exports via make command
}
type WorkerConfig struct {
	CompanyRequestTimeout time.Duration `envconfig:"COMPANY_REQUEST_TIMEOUT" default:"5s"`
	RetryOperationDelay   time.Duration `envconfig:"WORKER_RETRY_OPERATION_DELAY" default:"1s"`
}
type ServerConfig struct {
	Addr              string        `envconfig:"SERVER_ADDR" default:":8080"`
	ReadHeaderTimeout time.Duration `envconfig:"SERVER_READ_HEADER_TIMEOUT" default:"5s"`
	ShutdownTimeout   time.Duration `envconfig:"SERVER_SHUTDOWN_TIMEOUT" default:"5s"`
}
type KafkaConfig struct {
	Brokers           []string      `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`
	WriterMaxAttempts int           `envconfig:"KAFKA_WRITER_MAX_ATTEMPTS" default:"3"`
	WriterTimeout     time.Duration `envconfig:"KAFKA_WRITER_TIMEOUT" default:"5s"`
}
type DatabaseConfig struct {
	User     string        `envconfig:"POSTGRES_USER" default:"postgres"`
	Password string        `envconfig:"POSTGRES_PASSWORD" default:"password"`
	Name     string        `envconfig:"POSTGRES_DB" default:"postgres"`
	Host     string        `envconfig:"POSTGRES_HOST" default:"localhost"`
	Port     string        `envconfig:"POSTGRES_PORT" default:"5432"`
	Timeout  time.Duration `envconfig:"DATABASE_TIMEOUT" default:"5s"`
	SSLMode  string        `envconfig:"DATABASE_SSL_MODE" default:"disable"`
}

func NewConfig() (*Config, error) {

	config := Config{}
	err := envconfig.Process("", &config)
	if err != nil {
		return &Config{}, err
	}

	return &config, nil
}
func NewConfigMust() *Config {
	config, err := NewConfig()
	if err != nil {
		panic(err)
	}
	return config
}
