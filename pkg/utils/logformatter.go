package utils

import (
	"os"

	logger "github.com/sirupsen/logrus"
)

type Formatter struct {
	Fields           logger.Fields
	BuiltinFormatter logger.Formatter
}

func (f *Formatter) Format(entry *logger.Entry) ([]byte, error) {
	for k, v := range f.Fields {
		entry.Data[k] = v
	}
	return f.BuiltinFormatter.Format(entry)
}

func ConfigureLogger(eventID, keptnContext string, logLevel string) {
	logger.SetFormatter(&Formatter{
		Fields: logger.Fields{
			"service":      "splunk-service",
			"eventId":      eventID,
			"keptnContext": keptnContext,
		},
		BuiltinFormatter: &logger.TextFormatter{},
	})

	if os.Getenv(logLevel) != "" {
		logLevel, err := logger.ParseLevel(os.Getenv(logLevel))
		if err != nil {
			logger.WithError(err).Error("could not parse log level provided by 'LOG_LEVEL' env var")
		} else {
			logger.SetLevel(logLevel)
		}
	}
}
