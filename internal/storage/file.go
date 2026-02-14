package file

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
)

type Storage struct {
	filepath string
	logger   *slog.Logger
}

func New(logger *slog.Logger, filepath string) *Storage {
	return &Storage{
		filepath: filepath,
		logger:   logger,
	}
}

func (s *Storage) Save(ctx context.Context, data map[string]string) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		s.logger.Error("json marshal error", "error", err)
		return err
	}

	err = os.WriteFile(s.filepath, jsonData, 0644)
	if err != nil {
		s.logger.Error("write file error", "error", err)
		return err
	}

	return nil
}

func (s *Storage) Load(ctx context.Context) (map[string]string, error) {
	data, err := os.ReadFile(s.filepath)
	if err != nil {
		if os.IsNotExist(err) {
			s.logger.Info("storage file does not exist, starting with empty data")
			return make(map[string]string), nil
		}
		s.logger.Error("read file error", "error", err)
		return nil, err
	}

	if len(data) == 0 {
		return make(map[string]string), nil
	}

	var urlMap map[string]string
	err = json.Unmarshal(data, &urlMap)
	if err != nil {
		s.logger.Error("json unmarshal error", "error", err)
		return nil, err
	}

	return urlMap, nil
}
