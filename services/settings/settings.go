package settings

import (
	"context"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/rapid-downloader/rapid/api"
	"go.uber.org/fx"
)

const DataDir = ".rapid"

type SettingService struct {
	Path    string
	Setting api.Setting
}

func NewService(lc fx.Lifecycle) (*SettingService, api.Setting) {
	defaultSetting := defaultSetting()
	path := filepath.Join(defaultSetting.DataLocation, "settings.toml")

	svc := &SettingService{
		Path: path,
	}

	lc.Append(fx.StartHook(svc.Init))
	return svc, defaultSetting
}

func defaultSetting() api.Setting {
	home, _ := os.UserHomeDir()

	data := filepath.Join(home, DataDir)
	chunkLocation := filepath.Join(data, "chunks")
	logLocation := filepath.Join(data, "logs")

	return api.Setting{
		DownloadLocation: filepath.Join(home, "Downloads"),
		DataLocation:     data,
		ChunkLocation:    chunkLocation,
		LogLocation:      logLocation,
		MaxRetry:         3,
		MinChunkSize:     1024 * 1024 * 5, // 5 MB
		MaxChunkCount:    8,
	}
}

func (s *SettingService) Init(ctx context.Context) error {
	setting := defaultSetting()
	if err := os.MkdirAll(setting.DataLocation, os.ModePerm); err != nil {
		return err
	}

	if err := os.MkdirAll(setting.ChunkLocation, os.ModePerm); err != nil {
		return err
	}

	if _, err := os.Stat(s.Path); os.IsNotExist(err) {
		f, err := os.Create(s.Path)
		if err != nil {
			return err
		}
		defer f.Close()

		err = toml.NewEncoder(f).Encode(setting)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *SettingService) Load() (api.Setting, error) {
	var setting api.Setting
	_, err := toml.DecodeFile(s.Path, &setting)
	if err != nil {
		return api.Setting{}, err
	}

	return setting, nil
}
