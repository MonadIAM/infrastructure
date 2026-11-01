//go:build integration

package integration

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[Infrastructure] - Redis", Label("integration", "redis"), func() {
	for index, service := range services {
		Describe("[Service] - "+service, func() {
			Describe("[Behavior] - access and replication", func() {
				It("[case] - discovers one primary, two replicas and three Sentinel voters", func(ctx SpecContext) {
					// Arrange
					client := fixture.redis(ctx, service)
					marker := fixture.name + "/" + service

					// Act
					pods, podsErr := fixture.pods(ctx, "app.kubernetes.io/instance="+service+"-redis")
					writeErr := client.seed(ctx, marker, 2)

					// Assert
					Expect(podsErr).NotTo(HaveOccurred())
					Expect(pods).To(HaveLen(3))
					nodes := make(map[string]bool)
					for _, pod := range pods {
						nodes[pod.Spec.NodeName] = true
					}

					Expect(nodes).To(HaveLen(3))
					Expect(writeErr).NotTo(HaveOccurred())
					Eventually(func() error { return client.topology(ctx) }).WithContext(ctx).WithTimeout(time.Minute).Should(Succeed())
					Expect(client.replicated(ctx, marker)).To(Succeed())
				}, SpecTimeout(3*time.Minute))

				It("[case] - rejects foreign credentials, plaintext and missing client certificates", func(ctx SpecContext) {
					// Arrange
					client := fixture.redis(ctx, service)
					other := fixture.secret(ctx, services[(index+1)%len(services)]+"-redis-auth")
					foreign := *client
					foreign.password = string(other["redis-password"])
					plaintext := *client
					plaintext.certificate = nil
					anonymous := fixture.redis(ctx, service)
					anonymous.certificate.Certificates = nil
					var authenticationErrors, plaintextErrors, certificateErrors []error

					ports := []string{"6379", "26379"}
					for _, port := range ports {
						control := client.node(client.hosts[0], port)
						DeferCleanup(func() { _ = control.Close() })
						Expect(control.Ping(ctx).Err()).To(Succeed())
					}

					// Act
					for _, port := range ports {
						wrong := foreign.node(client.hosts[0], port)
						authenticationErrors = append(authenticationErrors, wrong.Ping(ctx).Err())
						DeferCleanup(func() { _ = wrong.Close() })
						insecure := plaintext.node(client.hosts[0], port)
						plaintextErrors = append(plaintextErrors, insecure.Ping(ctx).Err())
						DeferCleanup(func() { _ = insecure.Close() })
						missing := anonymous.node(client.hosts[0], port)
						certificateErrors = append(certificateErrors, missing.Ping(ctx).Err())
						DeferCleanup(func() { _ = missing.Close() })
					}

					// Assert
					Expect(authenticationErrors).To(HaveEach(MatchError(ContainSubstring("WRONGPASS"))))
					Expect(plaintextErrors).To(HaveEach(MatchError(MatchRegexp("EOF|connection reset by peer"))))
					Expect(certificateErrors).To(HaveEach(MatchError(MatchRegexp("certificate required|bad certificate|EOF|connection reset by peer"))))
				}, SpecTimeout(3*time.Minute))
			})

			Describe("[Behavior] - recovery", Label("failover"), func() {
				DescribeTable("[Behavior] - restores quorum, reconnects and preserves replicated values", func(ctx SpecContext, primaryFailure bool) {
					// Arrange
					client := fixture.redis(ctx, service)
					Eventually(func() error { return client.topology(ctx) }).WithContext(ctx).WithTimeout(time.Minute).Should(Succeed())
					oldPrimary, err := client.master(ctx)
					Expect(err).NotTo(HaveOccurred())
					failedHost := oldPrimary
					if !primaryFailure {
						for _, host := range client.hosts {
							if host != oldPrimary {
								failedHost = host
								break
							}
						}
					}

					observer := *client
					observer.hosts = nil
					for _, host := range client.hosts {
						if host != failedHost {
							observer.hosts = append(observer.hosts, host)
						}
					}

					marker := fmt.Sprintf("%s/%s/%t", fixture.name, service, primaryFailure)
					Expect(client.seed(ctx, marker, 2)).To(Succeed())
					Expect(client.replicated(ctx, marker)).To(Succeed())

					// Act
					restore := fixture.interruptPod(ctx, strings.Split(failedHost, ".")[0], "app.kubernetes.io/instance="+service+"-redis", 3)
					if primaryFailure {
						Eventually(func() (string, error) { return observer.master(ctx) }).WithContext(ctx).WithTimeout(7 * time.Minute).WithPolling(time.Second).Should(And(Not(BeEmpty()), Not(Equal(oldPrimary))))
					}

					Eventually(func() (string, error) { return client.client.Get(ctx, "integration:probe").Result() }).WithContext(ctx).WithTimeout(time.Minute).Should(Equal(marker))
					Eventually(func() error { return client.seed(ctx, marker+"/recovered", 1) }).WithContext(ctx).WithTimeout(time.Minute).Should(Succeed())
					current, masterErr := client.master(ctx)
					restore(ctx)
					recovered := fixture.redis(ctx, service)

					// Assert
					Expect(masterErr).NotTo(HaveOccurred())
					if primaryFailure {
						Expect(current).NotTo(Equal(oldPrimary))
					} else {
						Expect(current).To(Equal(oldPrimary))
					}

					Eventually(func() error { return recovered.topology(ctx) }).WithContext(ctx).WithTimeout(3 * time.Minute).Should(Succeed())
					Eventually(func() error { return recovered.replicated(ctx, marker+"/recovered") }).WithContext(ctx).WithTimeout(time.Minute).Should(Succeed())
				},
					Entry("[case] - primary failure", SpecTimeout(10*time.Minute), true),
					Entry("[case] - replica failure", SpecTimeout(10*time.Minute), false),
				)
			})
		})
	}
})
