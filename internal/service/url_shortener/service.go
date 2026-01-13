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
}

func New(repository Repository) *UrlService {
	return &UrlService{Repository: repository}
}

func (u *UrlService) Shorten(ctx context.Context, url string) (string, error) {
	if !isURLValid(url) {
		return "", ErrorInvalidURL
	}

	// возможно вынести в еще один сервисный слой, для удобства тестирования
	hash := md5.Sum([]byte(url))
	shortURL := fmt.Sprintf("%x", hash)[:8]

	err := u.Repository.SaveURLPair(ctx, url, shortURL)
	if err != nil {
		return "", err
	}

	return shortURL, nil
}

func (u *UrlService) Expand(ctx context.Context, shortURL string) (string, error) {
	URL, err := u.Repository.GetOriginURL(ctx, shortURL)
	if err != nil {
		return URL, err
	}

	return URL, nil
}

func isURLValid(url string) bool {
	re := regexp.MustCompile(URLPattern)
	return re.MatchString(url)
}
