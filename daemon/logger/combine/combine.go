// +build linux

// Package combine provides the log driver for forwarding server logs to
// multiple log drivers
package combine

import (
    "strings"

    "github.com/Sirupsen/logrus"
    "github.com/docker/docker/daemon/logger"
)

const name = "combine"

type combineLogger struct {
    loggers map[string]logger.Logger
    reader  string
}

func init() {
    if err := logger.RegisterLogDriver(name, New); err != nil {
        logrus.Fatal(err)
    }
    if err := logger.RegisterLogOptValidator(name, ValidateLogOpt); err != nil {
        logrus.Fatal(err)
    }
}

func New(ctx logger.Context) (logger.Logger, error) {
    combine := &combineLogger{}
    combine.loggers = make(map[string]logger.Logger)
    drivers, err := parseDrivers(ctx.Config["combine-drivers"])
    if err != nil {
        return nil, err
    }

    combine.reader = ctx.Config["combine-reader"]
    for _, name := range drivers {
        logdriver, err := logger.GetLogDriver(name)
        if err != nil {
            return nil, err
        }

        combine.loggers[name], err = logdriver(ctx)
        if err != nil {
            return nil, err
        }
    }

    return combine, nil
}

func (s *combineLogger) Log(msg *logger.Message) error {
    for _, value := range s.loggers {
        err := value.Log(msg)
        if err != nil {
            return err
        }
    }

    return nil
}

func (s *combineLogger) Close() error {
    for _, value := range s.loggers {
        err := value.Close()
        if err != nil {
            return err
        }
    }

    return nil
}

func (s *combineLogger) Name() string {
    return name
}

func (s *combineLogger) ReadLogs(config logger.ReadConfig) *logger.LogWatcher {
    if s.reader != "" {
        cLog := s.loggers[s.reader]
        logReader, ok := cLog.(logger.LogReader)
        if !ok {
            return logger.NewLogWatcher()
        }

        return logReader.ReadLogs(config)
    }

    return logger.NewLogWatcher()
}

func (s *combineLogger) LogPath() string {
    cLog := s.loggers[s.reader]
    logReader, ok := cLog.(logger.LogReader)
    if !ok {
        return ""
    }

    return logReader.LogPath()
}

func ValidateLogOpt(cfg map[string]string) error {
    if _, err := parseDrivers(cfg["combine-drivers"]); err != nil {
        return err
    }

    return nil
}

func parseDrivers(drivers string) ([]string, error) {
    return strings.Split(drivers, ","), nil
}
