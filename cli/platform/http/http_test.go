package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHTTP(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HTTP transport")
}

var _ = Describe("[Utility] - HTTP", func() {
	Describe("[Method] - Client.Do", func() {
		It("[case] - does not disclose server responses or follow redirects with credentials", func() {
			// Arrange
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				requests.Add(1)
				writer.Header().Set("Location", "/private-token")
				writer.WriteHeader(http.StatusTemporaryRedirect)
				_, _ = io.WriteString(writer, "private-secret")
			}))
			DeferCleanup(server.Close)
			client := New(server.URL)
			client.Token = "private-token"

			// Act
			_, err := client.Do(context.Background(), http.MethodGet, "/", nil)

			// Assert
			Expect(IsStatus(err, 307)).To(BeTrue())
			Expect(err.Error()).NotTo(ContainSubstring("private"))
			Expect(requests.Load()).To(Equal(int32(1)))
		})
	})

	Describe("[Function] - Wait", func() {
		It("[case] - stops a retry wait when the context is canceled", func() {
			// Arrange
			ctx, cancel := context.WithCancel(context.Background())
			DeferCleanup(cancel)
			result := make(chan error, 1)

			// Act
			go func() {
				result <- Wait(ctx, time.Hour, func() error {
					cancel()

					return StatusError{Code: 503}
				})
			}()

			var actionErr error
			select {
			case actionErr = <-result:
			case <-time.After(time.Second):
				actionErr = errors.New("cancellation did not finish within one second")
			}

			// Assert
			Expect(actionErr).To(MatchError(context.Canceled))
		})
	})
})
