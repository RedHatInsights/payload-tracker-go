package logging

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/redhatinsights/payload-tracker-go/internal/config"
	lc "github.com/redhatinsights/platform-go-middlewares/v2/logging/cloudwatch"
	"github.com/sirupsen/logrus"
)

// CustomCloudwatch adds hostname and app name
type CustomCloudwatch struct {
	Hostname string
}

// Marshaler is an interface any type can implement to change its output in our production logs.
type Marshaler interface {
	MarshalLog() map[string]interface{}
}

// Log is an instance of the global logrus.Logger
var Log *logrus.Logger
var logLevel logrus.Level

// NewCloudwatchFormatter creates a new logrus formatter for cloudwatch
func NewCloudwatchFormatter(cfg *config.TrackerConfig) *CustomCloudwatch {
	f := &CustomCloudwatch{
		Hostname: cfg.Hostname,
	}

	if f.Hostname == "" {
		f.Hostname = "unknown"
	}

	return f
}

// Format is the log formatter for the entry
func (f *CustomCloudwatch) Format(entry *logrus.Entry) ([]byte, error) {
	b := &bytes.Buffer{}

	now := time.Now()

	data := map[string]interface{}{
		"@timestamp":  now.Format("2006-01-02T15:04:05.999Z"),
		"@version":    1,
		"message":     entry.Message,
		"levelname":   entry.Level.String(),
		"source_host": f.Hostname,
		"app":         "payload-tracker",
		"caller":      entry.Caller.Func.Name(),
	}

	for k, v := range entry.Data {
		switch v := v.(type) {
		case error:
			data[k] = v.Error()
		case Marshaler:
			data[k] = v.MarshalLog()
		default:
			data[k] = v
		}
	}

	j, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	// Add newline to make stdout readable
	j = append(j, 0x0a)

	b.Write(j)

	return b.Bytes(), nil
}

// InitLogger initializes the global logger
func InitLogger() *logrus.Logger {

	cfg := config.Get()
	key := cfg.CloudwatchConfig.CWAccessKey
	secret := cfg.CloudwatchConfig.CWSecretKey
	region := cfg.CloudwatchConfig.CWRegion
	group := cfg.CloudwatchConfig.CWLogGroup
	stream := cfg.Hostname

	switch cfg.LogLevel {
	case "DEBUG":
		logLevel = logrus.DebugLevel
	case "ERROR":
		logLevel = logrus.ErrorLevel
	default:
		logLevel = logrus.InfoLevel
	}
	if flag.Lookup("test.v") != nil {
		logLevel = logrus.FatalLevel
	}

	formatter := NewCloudwatchFormatter(cfg)

	Log = &logrus.Logger{
		Out:          os.Stdout,
		Level:        logLevel,
		Formatter:    formatter,
		Hooks:        make(logrus.LevelHooks),
		ReportCaller: true,
	}

	if key != "" {
		cred := credentials.NewStaticCredentials(key, secret, "")
		awsconf := aws.NewConfig().WithRegion(region).WithCredentials(cred)
		batchWriter, err := lc.NewBatchWriterWithDuration(group, stream, awsconf, 10*time.Second)
		if err != nil {
			Log.Info(err)
		}
		hook := lc.NewLogrusHook(batchWriter)
		Log.Hooks.Add(hook)
	}

	return Log
}
