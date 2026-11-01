//go:build integration

package integration

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

type postgresqlClient struct {
	config     *pgx.ConnConfig
	connection *pgx.Conn
}

func (client *postgresqlClient) connect(ctx context.Context) (*pgx.Conn, error) {
	if client.connection != nil && !client.connection.IsClosed() {
		return client.connection, nil
	}

	connection, err := pgx.ConnectConfig(ctx, client.config.Copy())
	var failure *pgconn.ConnectError
	if errors.As(err, &failure) {
		return connection, fmt.Errorf("integration.postgresqlClient.connect: %w", failure.Unwrap())
	}

	client.connection = connection

	return connection, err
}

func (client *postgresqlClient) seed(ctx context.Context, value string) error {
	connection, err := client.connect(ctx)
	if err != nil {
		return err
	}

	_, err = connection.Exec(ctx, "CREATE TABLE IF NOT EXISTS integration_probe (id integer PRIMARY KEY, value text NOT NULL)")
	if err != nil {
		return err
	}

	_, err = connection.Exec(ctx, "INSERT INTO integration_probe VALUES (1, $1) ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value", value)

	return err
}

func (client *postgresqlClient) read(ctx context.Context) (string, error) {
	connection, err := client.connect(ctx)
	if err != nil {
		return "", err
	}

	var value string
	err = connection.QueryRow(ctx, "SELECT value FROM integration_probe WHERE id = 1").Scan(&value)

	return value, err
}

func (fixture *clusterFixture) postgresql(ctx context.Context, service, target string) *postgresqlClient {
	ginkgo.GinkgoHelper()
	cluster := service + "-postgresql"
	secret := fixture.secret(ctx, cluster+"-app")
	certificate := fixture.secret(ctx, cluster+"-ca")
	authorities := x509.NewCertPool()
	gomega.Expect(authorities.AppendCertsFromPEM(certificate["ca.crt"])).To(gomega.BeTrue())
	address := fixture.forward(ctx, target, "5432")
	host, port, err := net.SplitHostPort(address)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	parsedPort, err := strconv.ParseUint(port, 10, 16)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	configuration, err := pgx.ParseConfig("sslmode=verify-full")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	configuration.Host = host
	configuration.Port = uint16(parsedPort)
	configuration.Database = service + "-service"
	configuration.User = string(secret["username"])
	configuration.Password = string(secret["password"])
	configuration.ConnectTimeout = 5 * time.Second
	configuration.Fallbacks = nil
	configuration.TLSConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    authorities,
		ServerName: cluster + "-rw.monadiam-data.svc",
	}

	client := &postgresqlClient{config: configuration}
	ginkgo.DeferCleanup(func(ctx context.Context) {
		if client.connection != nil {
			_ = client.connection.Close(ctx)
		}
	})

	return client
}

func (fixture *clusterFixture) postgresqlPrimary(ctx context.Context, service string) (string, error) {
	output, err := fixture.kubectl(ctx, "get", "cluster.postgresql.cnpg.io", service+"-postgresql", "-n", "monadiam-data", "-o", "jsonpath={.status.currentPrimary}")

	return strings.TrimSpace(output), err
}

func (fixture *clusterFixture) postgresqlValues(ctx context.Context) ([]string, error) {
	values := make([]string, 0, len(services))
	for _, service := range services {
		client := fixture.postgresql(ctx, service, "service/"+service+"-postgresql-rw")
		value, err := client.read(ctx)
		if err != nil {
			return nil, fmt.Errorf("integration.clusterFixture.postgresqlValues: %s: %w", service, err)
		}

		values = append(values, value)
	}

	return values, nil
}

func (fixture *clusterFixture) seed(ctx context.Context) error {
	for _, service := range services {
		client := fixture.postgresql(ctx, service, "service/"+service+"-postgresql-rw")
		err := client.seed(ctx, fixture.name)
		if err != nil {
			return err
		}
	}

	return nil
}
