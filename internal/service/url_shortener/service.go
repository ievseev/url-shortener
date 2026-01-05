package url_shortener

import (
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
	SaveURLPair(urlOrigin, urlShort string) error
	GetOriginURL(urlShort string) (string, error)
}

type UrlService struct {
	Repository Repository
}

func New(repository Repository) *UrlService {
	return &UrlService{Repository: repository}
}

func (u *UrlService) Shorten(url string) (string, error) {
	if !isURLValid(url) {
		return "", ErrorInvalidURL
	}

	// возможно вынести в еще один сервисный слой, для удобства тестирования
	hash := md5.Sum([]byte(url))
	shortURL := fmt.Sprintf("%x", hash)[:8]

	err := u.Repository.SaveURLPair(url, shortURL)
	if err != nil {
		return "", err
	}

	return shortURL, nil
}

func (u *UrlService) Expand(shortURL string) (string, error) {
	URL, err := u.Repository.GetOriginURL(shortURL)
	if err != nil {
		return URL, err
	}

	return URL, nil
}

func isURLValid(url string) bool {
	re := regexp.MustCompile(URLPattern)
	return re.MatchString(url)
}
