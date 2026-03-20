package urlshortener

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"regexp"

	"github.com/ievseev/url-shortener/internal/model"
	urlrepo "github.com/ievseev/url-shortener/internal/repository/url"
)

const patternURL = `^https?://[^\s/$.?#].[^\s]*$`

var (
	ErrorInvalidURL   = errors.New("invalid url")
	ErrorURLConflict  = errors.New("url conflict")
	ErrorUnauthorized = errors.New("unauthorized")
)

type Repository interface {
	SaveURL(ctx context.Context, userID, urlOrigin, shortURLBase string) (string, error)
	SaveURLBatch(ctx context.Context, userID string, urlOrigins, shortURLBases []string) ([]string, error)
	GetOriginURL(ctx context.Context, urlShort string) (string, error)
	GetUserURLs(ctx context.Context, userID string) ([]model.UserURL, error)
	Ping(ctx context.Context) error
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

func (u *URLService) Shorten(ctx context.Context, userID, url string) (string, error) {
	if userID == "" {
		return "", ErrorUnauthorized
	}

	if !u.re.MatchString(url) {
		return "", fmt.Errorf("%w: %s", ErrorInvalidURL, url)
	}

	shortURL, err := u.Repository.SaveURL(ctx, userID, url, u.generateBaseShortURL(url))
	if err != nil {
		if errors.Is(err, urlrepo.ErrOriginalURLConflict) {
			return shortURL, errors.Join(ErrorURLConflict, err)
		}

		return "", fmt.Errorf("save url pair error: %w", err)
	}

	return shortURL, nil
}

func (u *URLService) ShortenBatch(ctx context.Context, userID string, urls []string) ([]string, error) {
	if userID == "" {
		return nil, ErrorUnauthorized
	}

	shortURLBases := make([]string, len(urls))

	for i, currentURL := range urls {
		if !u.re.MatchString(currentURL) {
			return nil, fmt.Errorf("%w: %s", ErrorInvalidURL, currentURL)
		}

		shortURLBases[i] = u.generateBaseShortURL(currentURL)
	}

	shortURLs, err := u.Repository.SaveURLBatch(ctx, userID, urls, shortURLBases)
	if err != nil {
		return nil, fmt.Errorf("save url batch error: %w", err)
	}

	return shortURLs, nil
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

func (u *URLService) GetUserURLs(ctx context.Context, userID string) ([]model.UserURL, error) {
	if userID == "" {
		return nil, ErrorUnauthorized
	}

	userURLs, err := u.Repository.GetUserURLs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user urls error: %w", err)
	}

	return userURLs, nil
}
