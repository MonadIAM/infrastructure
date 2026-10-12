package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHTTP(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HTTP transport")
}

var _ = Describe("[Utility] - HTTP", func() {
	Describe("[Function] - New", func() {
		It("[case] - returns the redirect response without forwarding credentials", func() {
			// Arrange
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				requests.Add(1)
				writer.Header().Set("Location", "/private-token")
				writer.WriteHeader(http.StatusTemporaryRedirect)
			}))
			DeferCleanup(server.Close)
			client := New(server.URL)
			client.SetHeader("X-Vault-Token", "private-token")

			// Act
			response, err := client.R().Get("/")

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode()).To(Equal(http.StatusTemporaryRedirect))
			Expect(requests.Load()).To(Equal(int32(1)))
		})

		It("[case] - limits responses to 16 MiB", func() {
			// Arrange
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				chunk := make([]byte, 1<<20)
				for range 17 {
					_, err := writer.Write(chunk)
					if err != nil {
						return
					}
				}
			}))
			DeferCleanup(server.Close)
			client := New(server.URL)

			// Act
			_, err := client.R().Get("/")

			// Assert
			Expect(err).To(MatchError(resty.ErrResponseBodyTooLarge))
		})
	})

	Describe("[Method] - RequestError.Error", func() {
		It("[case] - preserves cancellation and its cause without exposing the URL", func() {
			// Arrange
			cause := &url.Error{Op: "Get", URL: "http://host/private-secret", Err: context.Canceled}

			// Act
			err := RequestError{Cause: cause}

			// Assert
			Expect(err).To(MatchError(context.Canceled))
			Expect(errors.Is(err, cause)).To(BeTrue())
			Expect(err.Error()).NotTo(ContainSubstring("private-secret"))
		})
	})
})
