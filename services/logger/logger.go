package logger

import (
	"bufio"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rapid-downloader/rapid/api"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var once sync.Once

type LoggerService struct {
	setting api.Setting
	log     *zap.Logger
	fs      *os.File
	mutex   sync.Mutex
}

func NewService(lc fx.Lifecycle, setting api.Setting) *LoggerService {
	svc := &LoggerService{
		setting: setting,
		mutex:   sync.Mutex{},
	}

	lc.Append(fx.Hook{
		OnStart: svc.Init,
		OnStop:  svc.Close,
	})

	return svc
}

// Init initializes the write only today's logger file
func (s *LoggerService) Init(ctx context.Context) error {
	var err error
	once.Do(func() {
		now := time.Now().Format(time.DateOnly)
		filename := filepath.Join(s.setting.LogLocation, now)
		if err := os.MkdirAll(s.setting.LogLocation, os.ModePerm); err != nil {
			s.Error(err)
		}

		s.fs, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			s.Error(err)
		}
	})

	return err
}

// Close closes the logger file
func (s *LoggerService) Close(ctx context.Context) error {
	if s.fs != nil {
		return s.fs.Close()
	}
	return nil
}

func (s *LoggerService) Error(err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.fs != nil {
		s.fs.WriteString(err.Error())
	}
}

var ErrNotInitialized = errors.New("logger service not initialized")

func (s *LoggerService) GetLogs(date string) ([]string, error) {
	if date == "" {
		date = time.Now().Format(time.DateOnly)
	}

	if s.fs == nil {
		return nil, ErrNotInitialized
	}

	filename := filepath.Join(s.setting.LogLocation, date)
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	logEntries := make([]string, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		logEntries = append(logEntries, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return logEntries, nil
}
