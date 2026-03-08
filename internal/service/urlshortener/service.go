package urlshortener

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"regexp"

	URLRepo "github.com/ievseev/url-shortener/internal/repository/url"
)

const patternURL = `^https?://[^\s/$.?#].[^\s]*$`

var (
	ErrorInvalidURL = errors.New("invalid url")
)

type Repository interface {
	SaveURLPair(ctx context.Context, urlOrigin, urlShort string) error
	GetOriginURL(ctx context.Context, urlShort string) (string, error)
}

type URLService struct {
	Repository Repository
	re         *regexp.Regexp
}

func New(repository Repository) (*URLService, error) {
	re, err := regexp.Compile(patternURL)
	if err != nil {
		return nil, fmt.Errorf("regexp compile error: %w", err)
	}

	return &URLService{Repository: repository, re: re}, nil
}

func (u *URLService) Shorten(ctx context.Context, url string) (string, error) {
	if !u.re.MatchString(url) {
		return "", fmt.Errorf("%w: %s", ErrorInvalidURL, url)
	}

	// возможно вынести в еще один сервисный слой, для удобства тестирования
	baseShortURL := u.generateBaseShortURL(url)
	shortURL := baseShortURL

	// Обработка коллизий
	counter := 0
	for {
		existingURL, err := u.Repository.GetOriginURL(ctx, shortURL)
		if errors.Is(err, URLRepo.ErrOriginURLNotFound) {
			// значит наш shortURL оригинален, коллизии нет
			break
		}
		if err != nil {
			return "", fmt.Errorf("check short url collision error: %w", err)
		}
		if existingURL == url {
			break
		}

		// Если коллизия, добавляем суффикс
		counter++
		shortURL = fmt.Sprintf("%s_%d", baseShortURL, counter)
	}

	err := u.Repository.SaveURLPair(ctx, url, shortURL)
	if err != nil {
		return "", fmt.Errorf("save url pair error: %w", err)
	}

	return shortURL, nil
}

func (u *URLService) generateBaseShortURL(url string) string {
	hash := md5.Sum([]byte(url))
	return fmt.Sprintf("%x", hash)[:8]
}

func (u *URLService) Expand(ctx context.Context, shortURL string) (string, error) {
	URL, err := u.Repository.GetOriginURL(ctx, shortURL)
	if err != nil {
		return URL, fmt.Errorf("get origin url error: %w", err)
	}

	return URL, nil
}
