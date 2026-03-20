package file

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"

	urlrepo "github.com/ievseev/url-shortener/internal/repository/url"
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

func (s *Storage) Save(ctx context.Context, snapshot urlrepo.Snapshot) error {
	jsonData, err := json.Marshal(snapshot)
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

func (s *Storage) Load(ctx context.Context) (urlrepo.Snapshot, error) {
	data, err := os.ReadFile(s.filepath)
	if err != nil {
		if os.IsNotExist(err) {
			s.logger.Info("storage file does not exist, starting with empty data")
			return urlrepo.Snapshot{
				URLs:     make(map[string]string),
				UserURLs: make(map[string][]string),
			}, nil
		}
		s.logger.Error("read file error", "error", err)
		return urlrepo.Snapshot{}, err
	}

	if len(data) == 0 {
		return urlrepo.Snapshot{
			URLs:     make(map[string]string),
			UserURLs: make(map[string][]string),
		}, nil
	}

	var snapshot urlrepo.Snapshot
	if err := json.Unmarshal(data, &snapshot); err == nil && (snapshot.URLs != nil || snapshot.UserURLs != nil) {
		if snapshot.URLs == nil {
			snapshot.URLs = make(map[string]string)
		}
		if snapshot.UserURLs == nil {
			snapshot.UserURLs = make(map[string][]string)
		}

		return snapshot, nil
	}

	var legacyURLMap map[string]string
	if err := json.Unmarshal(data, &legacyURLMap); err != nil {
		s.logger.Error("json unmarshal error", "error", err)
		return urlrepo.Snapshot{}, err
	}

	return urlrepo.Snapshot{
		URLs:     legacyURLMap,
		UserURLs: make(map[string][]string),
	}, nil
}
