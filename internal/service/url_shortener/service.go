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
	GetURLOrigin(urlShort string) (string, error)
}

type UrlShortener struct {
	Repository Repository
}

func New(repository Repository) *UrlShortener {
	return &UrlShortener{Repository: repository}
}

func (u *UrlShortener) Create(url string) (string, error) {
	var shortURL string

	if !isURLValid(url) {
		return shortURL, ErrorInvalidURL
	}

	// возможно вынести в еще один сервисный слой, для удобства тестирования
	hash := md5.Sum([]byte(url))
	shortURL = fmt.Sprintf("%x", hash)[:8]

	err := u.Repository.SaveURLPair(url, shortURL)
	if err != nil {
		return shortURL, err
	}

	return shortURL, nil
}

func isURLValid(url string) bool {
	re := regexp.MustCompile(URLPattern)
	return re.MatchString(url)
}
