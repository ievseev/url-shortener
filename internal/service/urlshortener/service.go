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
	ErrorInvalidURL  = errors.New("invalid url")
	ErrorURLConflict = errors.New("url conflict")
)

type Repository interface {
	SaveURL(ctx context.Context, urlOrigin, shortURLBase string) (string, error)
	SaveURLBatch(ctx context.Context, urlOrigins, shortURLBases []string) ([]string, error)
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

	shortURL, err := u.Repository.SaveURL(ctx, url, u.generateBaseShortURL(url))
	if err != nil {
		if errors.Is(err, URLRepo.ErrOriginalURLConflict) {
			return shortURL, errors.Join(ErrorURLConflict, err)
		}

		return "", fmt.Errorf("save url pair error: %w", err)
	}

	return shortURL, nil
}

func (u *URLService) ShortenBatch(ctx context.Context, urls []string) ([]string, error) {
	shortURLBases := make([]string, len(urls))

	for i, currentURL := range urls {
		if !u.re.MatchString(currentURL) {
			return nil, fmt.Errorf("%w: %s", ErrorInvalidURL, currentURL)
		}

		shortURLBases[i] = u.generateBaseShortURL(currentURL)
	}

	shortURLs, err := u.Repository.SaveURLBatch(ctx, urls, shortURLBases)
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
