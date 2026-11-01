//go:build integration

package integration

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[Infrastructure] - PostgreSQL", Label("integration", "postgresql"), func() {
	for index, service := range services {
		Describe("[Service] - "+service, func() {
			Describe("[Behavior] - access and replication", func() {
				It("[case] - provides two instances, dedicated volumes and application replication", func(ctx SpecContext) {
					// Arrange
					cluster := service + "-postgresql"
					primary := fixture.postgresql(ctx, service, "service/"+cluster+"-rw")
					replica := fixture.postgresql(ctx, service, "service/"+cluster+"-ro")
					marker := fixture.name + "/" + service

					// Act
					output, topologyErr := fixture.kubectl(ctx, "get", "cluster.postgresql.cnpg.io", cluster, "-n", "monadiam-data", "-o", "jsonpath={.spec.instances}/{.status.readyInstances}/{.spec.storage.size}/{.spec.walStorage.size}")
					pods, podsErr := fixture.pods(ctx, "cnpg.io/cluster="+cluster+",cnpg.io/podRole=instance")
					state, stateErr := fixture.state(ctx)
					volumes, volumesErr := fixture.kubectl(ctx, "get", "pvc", "-n", "monadiam-data", "-l", "cnpg.io/cluster="+cluster, "-o", `jsonpath={range .items[*]}{.spec.resources.requests.storage}/{.status.phase}{" "}{end}`)
					writeErr := primary.seed(ctx, marker)

					// Assert
					Expect(topologyErr).NotTo(HaveOccurred())
					Expect(output).To(Equal("2/2/8Gi/1Gi"))
					Expect(podsErr).NotTo(HaveOccurred())
					Expect(pods).To(HaveLen(2))
					Expect(stateErr).NotTo(HaveOccurred())
					Expect(volumesErr).NotTo(HaveOccurred())
					Expect(strings.Fields(volumes)).To(ConsistOf("8Gi/Bound", "1Gi/Bound", "8Gi/Bound", "1Gi/Bound"))
					for _, pod := range pods {
						claim := "monadiam-data/" + pod.Metadata.Name
						Expect(state.Volumes).To(HaveKey(claim))
						Expect(state.Volumes).To(HaveKey(claim + "-wal"))
					}

					Expect(state.Volumes).NotTo(HaveKey("monadiam-data/postgresql-main-1"))
					Expect(writeErr).NotTo(HaveOccurred())
					Eventually(func() (string, error) { return replica.read(ctx) }).WithContext(ctx).WithTimeout(time.Minute).WithPolling(time.Second).Should(Equal(marker))
				}, SpecTimeout(3*time.Minute))

				It("[case] - enforces application privileges, service credentials and TLS", func(ctx SpecContext) {
					// Arrange
					client := fixture.postgresql(ctx, service, "service/"+service+"-postgresql-rw")
					connection, err := client.connect(ctx)
					Expect(err).NotTo(HaveOccurred())
					other := fixture.secret(ctx, services[(index+1)%len(services)]+"-postgresql-app")
					foreign := fixture.postgresql(ctx, service, "service/"+service+"-postgresql-rw")
					foreign.config.User = string(other["username"])
					foreign.config.Password = string(other["password"])
					plaintext := fixture.postgresql(ctx, service, "service/"+service+"-postgresql-rw")
					plaintext.config.TLSConfig = nil
					standby := fixture.postgresql(ctx, service, "service/"+service+"-postgresql-ro")
					standbyConnection, err := standby.connect(ctx)
					Expect(err).NotTo(HaveOccurred())

					// Act
					var privileged, replication, tlsEnabled bool
					roleErr := connection.QueryRow(ctx, "SELECT rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication FROM pg_roles WHERE rolname = current_user").Scan(&privileged)
					debeziumErr := connection.QueryRow(ctx, "SELECT rolreplication AND NOT rolsuper AND pg_has_role('debezium', current_user, 'MEMBER') FROM pg_roles WHERE rolname = 'debezium'").Scan(&replication)
					tlsErr := connection.QueryRow(ctx, "SELECT ssl FROM pg_stat_ssl WHERE pid = pg_backend_pid()").Scan(&tlsEnabled)
					_, foreignErr := foreign.connect(ctx)
					_, plaintextErr := plaintext.connect(ctx)

					_, standbyErr := standbyConnection.Exec(ctx, "CREATE TABLE integration_readonly_probe (id integer)")

					// Assert
					Expect(roleErr).NotTo(HaveOccurred())
					Expect(privileged).To(BeFalse())
					Expect(debeziumErr).NotTo(HaveOccurred())
					Expect(replication).To(BeTrue())
					Expect(tlsErr).NotTo(HaveOccurred())
					Expect(tlsEnabled).To(BeTrue())
					var authenticationError *pgconn.PgError
					Expect(errors.As(foreignErr, &authenticationError)).To(BeTrue())
					Expect(authenticationError.Code).To(Equal("28P01"))
					Expect(errors.As(plaintextErr, &authenticationError)).To(BeTrue())
					Expect(authenticationError.Code).To(Equal("28000"))
					Expect(errors.As(standbyErr, &authenticationError)).To(BeTrue())
					Expect(authenticationError.Code).To(Equal("25006"))
				}, SpecTimeout(3*time.Minute))

			})

			Describe("[Behavior] - recovery", Label("failover"), func() {
				DescribeTable("[Behavior] - restores the topology and preserves replicated rows", func(ctx SpecContext, primaryFailure bool) {
					// Arrange
					cluster := service + "-postgresql"
					selector := "cnpg.io/cluster=" + cluster + ",cnpg.io/podRole=instance"
					oldPrimary, err := fixture.postgresqlPrimary(ctx, service)
					Expect(err).NotTo(HaveOccurred())
					pods, err := fixture.pods(ctx, selector)
					Expect(err).NotTo(HaveOccurred())
					Expect(pods).To(HaveLen(2))
					failedPod := oldPrimary
					if !primaryFailure {
						for _, pod := range pods {
							if pod.Metadata.Name != oldPrimary {
								failedPod = pod.Metadata.Name
							}
						}
					}

					marker := fmt.Sprintf("%s/%s/%t", fixture.name, service, primaryFailure)
					primary := fixture.postgresql(ctx, service, "service/"+cluster+"-rw")
					replica := fixture.postgresql(ctx, service, "service/"+cluster+"-ro")
					Expect(primary.seed(ctx, marker)).To(Succeed())
					Eventually(func() (string, error) { return replica.read(ctx) }).WithContext(ctx).WithTimeout(time.Minute).WithPolling(time.Second).Should(Equal(marker))

					// Act
					restore := fixture.interruptPod(ctx, failedPod, selector, 2)
					if primaryFailure {
						Eventually(func() (string, error) { return fixture.postgresqlPrimary(ctx, service) }).WithContext(ctx).WithTimeout(3 * time.Minute).Should(And(Not(BeEmpty()), Not(Equal(oldPrimary))))
					}

					current, err := fixture.postgresqlPrimary(ctx, service)
					Expect(err).NotTo(HaveOccurred())
					Eventually(func() (string, error) {
						return fixture.endpoint(ctx, cluster+"-rw")
					}).WithContext(ctx).WithTimeout(time.Minute).Should(Equal(current))
					recovered := fixture.postgresql(ctx, service, "service/"+cluster+"-rw")
					value, readErr := recovered.read(ctx)
					writeErr := recovered.seed(ctx, marker+"/recovered")
					restore(ctx)
					rejoined := fixture.postgresql(ctx, service, "service/"+cluster+"-ro")

					// Assert
					Expect(readErr).NotTo(HaveOccurred())
					Expect(value).To(Equal(marker))
					Expect(writeErr).NotTo(HaveOccurred())
					if !primaryFailure {
						Expect(current).To(Equal(oldPrimary))
					}

					Eventually(func() (string, error) { return rejoined.read(ctx) }).WithContext(ctx).WithTimeout(time.Minute).WithPolling(time.Second).Should(Equal(marker + "/recovered"))
				},
					Entry("[case] - primary failure", SpecTimeout(10*time.Minute), true),
					Entry("[case] - replica failure", SpecTimeout(10*time.Minute), false),
				)
			})
		})
	}
})
