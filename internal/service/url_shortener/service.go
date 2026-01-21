package url_shortener

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"regexp"
)

const URLPattern = `^https?://[^\s/$.?#].[^\s]*$`

var (
	ErrorInvalidURL = errors.New("invalid url")
)

type Repository interface {
	SaveURLPair(ctx context.Context, urlOrigin, urlShort string) error
	GetOriginURL(ctx context.Context, urlShort string) (string, error)
}

type UrlService struct {
	Repository Repository
	re         *regexp.Regexp
}

func New(repository Repository) (*UrlService, error) {
	re, err := regexp.Compile(URLPattern)
	if err != nil {
		return nil, fmt.Errorf("regexp compile error: %w", err)
	}

	return &UrlService{Repository: repository, re: re}, nil
}

func (u *UrlService) Shorten(ctx context.Context, url string) (string, error) {
	if !u.re.MatchString(url) {
		return "", fmt.Errorf("%w: %s", ErrorInvalidURL, url)
	}

	// возможно вынести в еще один сервисный слой, для удобства тестирования
	baseShortURL := u.generateBaseShortURL(url)
	shortURL := baseShortURL

	// Обработка коллизий
	counter := 0
	for {
		originUrl, err := u.Repository.GetOriginURL(ctx, shortURL)
		if err != nil {
			return "", fmt.Errorf("get origin url error: %w", err)
		}

		if originUrl == "" {
			// значит наш shortURL оригинален, коллизии нет
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

func (u *UrlService) generateBaseShortURL(url string) string {
	hash := md5.Sum([]byte(url))
	return fmt.Sprintf("%x", hash)[:8]
}

func (u *UrlService) Expand(ctx context.Context, shortURL string) (string, error) {
	URL, err := u.Repository.GetOriginURL(ctx, shortURL)
	if err != nil {
		return URL, fmt.Errorf("get origin url error: %w", err)
	}

	return URL, nil
}
