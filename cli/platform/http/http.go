package http

import (
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
)

type discardLogger struct{}

func New(address string) *resty.Client {
	return resty.New().
		SetBaseURL(address).
		SetTimeout(30 * time.Second).
		SetResponseBodyLimit(16 << 20).
		SetDisableWarn(true).
		SetLogger(discardLogger{}).
		SetRedirectPolicy(resty.RedirectPolicyFunc(func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}))
}

func (discardLogger) Errorf(string, ...any) {}
func (discardLogger) Warnf(string, ...any)  {}
func (discardLogger) Debugf(string, ...any) {}
