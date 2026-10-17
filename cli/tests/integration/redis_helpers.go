//go:build integration

package integration

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/redis/go-redis/v9"
)

type redisClient struct {
	name        string
	password    string
	certificate *tls.Config
	addresses   map[string]string
	hosts       []string
	client      *redis.Client
}

func (client *redisClient) dial(ctx context.Context, network, address string) (net.Conn, error) {
	local, ok := client.addresses[address]
	if !ok {
		return nil, fmt.Errorf("integration.redisClient.dial: unknown address %s", address)
	}

	dialer := &net.Dialer{Timeout: 3 * time.Second}
	if client.certificate == nil {
		return dialer.DialContext(ctx, network, local)
	}

	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	configuration := client.certificate.Clone()
	configuration.ServerName = host
	tlsDialer := &tls.Dialer{NetDialer: dialer, Config: configuration}

	return tlsDialer.DialContext(ctx, network, local)
}

func (client *redisClient) node(host, port string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: net.JoinHostPort(host, port), Password: client.password, Dialer: client.dial,
		Protocol: 2, MaxRetries: -1, DialTimeout: 3 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second,
	})
}

func (client *redisClient) sentinel(host string) *redis.SentinelClient {
	return redis.NewSentinelClient(&redis.Options{
		Addr: net.JoinHostPort(host, "26379"), Password: client.password, Dialer: client.dial,
		Protocol: 2, MaxRetries: -1, DialTimeout: 3 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second,
	})
}

func (client *redisClient) master(ctx context.Context) (string, error) {
	var lastErr error
	for _, host := range client.hosts {
		sentinel := client.sentinel(host)
		address, err := sentinel.GetMasterAddrByName(ctx, client.name).Result()
		_ = sentinel.Close()
		if err != nil {
			lastErr = err
			continue
		}

		if len(address) != 2 {
			return "", fmt.Errorf("integration.redisClient.master: invalid Sentinel reply for %s", client.name)
		}

		return address[0], nil
	}

	return "", lastErr
}

func (client *redisClient) topology(ctx context.Context) error {
	master, err := client.master(ctx)
	if err != nil {
		return err
	}

	masters := 0
	for _, host := range client.hosts {
		err := client.voter(ctx, host, master)
		if err != nil {
			return err
		}

		node := client.node(host, "6379")
		info, err := node.Info(ctx, "replication").Result()
		_ = node.Close()
		if err != nil {
			return err
		}

		if strings.Contains(info, "role:master\r\n") && host == master && strings.Contains(info, "connected_slaves:2\r\n") {
			masters++
		} else if !strings.Contains(info, "role:slave\r\n") || !strings.Contains(info, "master_link_status:up\r\n") {
			return fmt.Errorf("integration.redisClient.topology: unexpected replication state on %s", host)
		}
	}

	if masters != 1 {
		return fmt.Errorf("integration.redisClient.topology: expected one primary, got %d", masters)
	}

	return nil
}

func (client *redisClient) voter(ctx context.Context, host, master string) error {
	sentinel := client.sentinel(host)
	defer func() { _ = sentinel.Close() }()
	state, err := sentinel.Master(ctx, client.name).Result()
	if err != nil {
		return err
	}

	if state["ip"] != master || state["quorum"] != "2" || state["num-other-sentinels"] != "2" || state["num-slaves"] != "2" {
		return fmt.Errorf("integration.redisClient.voter: inconsistent Sentinel state on %s", host)
	}

	quorum, err := sentinel.CkQuorum(ctx, client.name).Result()
	if err != nil {
		return err
	}

	if !strings.HasPrefix(quorum, "OK 3 usable Sentinels") {
		return fmt.Errorf("integration.redisClient.voter: incomplete quorum on %s: %s", host, quorum)
	}

	return nil
}

func (client *redisClient) seed(ctx context.Context, value string, replicas int) error {
	connection := client.client.Conn()
	defer func() { _ = connection.Close() }()
	err := connection.Set(ctx, "integration:probe", value, 0).Err()
	if err != nil {
		return err
	}

	acknowledged, err := connection.Wait(ctx, replicas, 5*time.Second).Result()
	if err != nil {
		return err
	}

	if acknowledged < int64(replicas) {
		return fmt.Errorf("integration.redisClient.seed: %d replicas acknowledged, expected %d", acknowledged, replicas)
	}

	return nil
}

func (client *redisClient) replicated(ctx context.Context, expected string) error {
	for _, host := range client.hosts {
		node := client.node(host, "6379")
		value, err := node.Get(ctx, "integration:probe").Result()
		_ = node.Close()
		if err != nil {
			return err
		}

		if value != expected {
			return fmt.Errorf("integration.redisClient.replicated: unexpected value on %s", host)
		}
	}

	return nil
}

func (fixture *clusterFixture) redis(ctx context.Context, service string) *redisClient {
	ginkgo.GinkgoHelper()
	name := service + "-redis"
	secret := fixture.secret(ctx, name+"-auth")
	material := fixture.secret(ctx, name+"-tls")
	certificate, err := tls.X509KeyPair(material["tls.crt"], material["tls.key"])
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	authorities := x509.NewCertPool()
	gomega.Expect(authorities.AppendCertsFromPEM(material["ca.crt"])).To(gomega.BeTrue())
	client := &redisClient{
		name: name, password: string(secret["redis-password"]), addresses: make(map[string]string),
		certificate: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: authorities, Certificates: []tls.Certificate{certificate}},
	}
	var sentinels []string
	for index := 0; index < 3; index++ {
		pod := fmt.Sprintf("%s-node-%d", name, index)
		host := pod + "." + name + "-headless.monadiam-data.svc.cluster.local"
		client.hosts = append(client.hosts, host)
		for _, port := range []string{"6379", "26379"} {
			client.addresses[net.JoinHostPort(host, port)] = fixture.forward(ctx, "pod/"+pod, port)
		}

		sentinels = append(sentinels, net.JoinHostPort(host, "26379"))
	}

	client.client = redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName: name, SentinelAddrs: sentinels, Password: client.password, SentinelPassword: client.password,
		Dialer: client.dial, Protocol: 2, MaxRetries: -1, DialTimeout: 3 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second,
	})
	ginkgo.DeferCleanup(func() { _ = client.client.Close() })

	return client
}

func (fixture *clusterFixture) seedRedis(ctx context.Context) error {
	for _, service := range services {
		client := fixture.redis(ctx, service)
		err := client.seed(ctx, fixture.name, 2)
		if err != nil {
			return err
		}
	}

	return nil
}

func (fixture *clusterFixture) redisValues(ctx context.Context) ([]string, error) {
	values := make([]string, 0, len(services))
	for _, service := range services {
		client := fixture.redis(ctx, service)
		value, err := client.client.Get(ctx, "integration:probe").Result()
		if err != nil {
			return nil, err
		}

		values = append(values, value)
	}

	return values, nil
}
