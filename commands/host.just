set quiet
set working-directory := '..'

profile := env_var_or_default('PROFILE', 'dev')
kubeconfig := env_var('HOME') + '/.kube/monadiam-microk8s-' + profile + '.yaml'
context := 'monadiam-microk8s-' + profile

# Show IP addresses, free disk space, and required tools without changing the host.
inspect:
    ip -br -4 addr
    df -h /var/lib
    command -v just go terraform kubectl helm docker snap openssl mkfs.ext4

# Create empty Rawfile and Zot ext4 volumes on Ubuntu; fail if either image exists.
storage rawfile_size='320G' zot_size='32G':
    sudo test ! -e /var/lib/monadiam/microk8s-volumes.ext4
    sudo test ! -e /var/lib/monadiam/zot.ext4
    sudo install -d -m 700 \
        /var/lib/monadiam/microk8s/volumes \
        /var/lib/monadiam/microk8s/volume-metadata \
        /var/lib/monadiam/postgresql-wal
    sudo install -d -m 755 /var/lib/monadiam/zot
    sudo truncate -s '{{ rawfile_size }}' /var/lib/monadiam/microk8s-volumes.ext4
    sudo truncate -s '{{ zot_size }}' /var/lib/monadiam/zot.ext4
    sudo mkfs.ext4 -F /var/lib/monadiam/microk8s-volumes.ext4
    sudo mkfs.ext4 -F /var/lib/monadiam/zot.ext4
    sudo install -m 644 bootstrap/microk8s/var-lib-monadiam-microk8s-volumes.mount /etc/systemd/system/
    sudo install -m 644 bootstrap/zot/var-lib-monadiam-zot.mount /etc/systemd/system/
    sudo systemctl daemon-reload
    sudo systemctl enable --now var-lib-monadiam-microk8s-volumes.mount var-lib-monadiam-zot.mount
    sudo chmod 700 /var/lib/monadiam/microk8s/volumes
    sudo install -d -o 65534 -g 65534 -m 755 /var/lib/monadiam/zot/data

# Install MicroK8s on a new Ubuntu host with its IP and a host-only API encryption key.
install node_ip:
    @! snap list microk8s >/dev/null 2>&1
    sudo install -d -m 700 /etc/monadiam/microk8s
    sudo test ! -e /etc/monadiam/microk8s/encryption.key
    sudo sh -c 'umask 077; openssl rand -base64 32 > /etc/monadiam/microk8s/encryption.key'
    sudo install -m 600 /dev/null /etc/monadiam/microk8s/launch.yaml
    sed 's/192\.168\.1\.143/{{ node_ip }}/' \
        clusters/dev/host/microk8s.yaml | \
        sudo tee /etc/monadiam/microk8s/launch.yaml >/dev/null
    sudo install -d -m 755 /var/snap/microk8s/common
    sudo install -m 600 /dev/null /var/snap/microk8s/common/.microk8s.yaml
    sudo awk 'BEGIN { getline key < "/etc/monadiam/microk8s/encryption.key" } \
        { gsub(/@ENCRYPTION_KEY@/, key); print }' \
        /etc/monadiam/microk8s/launch.yaml | \
        sudo tee /var/snap/microk8s/common/.microk8s.yaml >/dev/null
    sudo snap install microk8s --classic --channel=1.35/stable
    sudo microk8s status --wait-ready

# Pin Calico IP detection to the specified network interface.
calico interface:
    sudo sed -i \
        '/- name: IP_AUTODETECTION_METHOD/{n;s/value: .*/value: "interface={{ interface }}"/;}' \
        /var/snap/microk8s/current/args/cni-network/cni.yaml
    sudo microk8s kubectl apply -f /var/snap/microk8s/current/args/cni-network/cni.yaml

# Apply MicroK8s systemd limits and Rawfile mount dependency, then restart MicroK8s.
limits:
    sudo install -m 644 clusters/dev/host/monadiam.slice /etc/systemd/system/
    sudo install -D -m 644 \
        bootstrap/microk8s/resources.conf \
        /etc/systemd/system/snap.microk8s.daemon-kubelite.service.d/resources.conf
    sudo install -D -m 644 \
        bootstrap/microk8s/resources.conf \
        /etc/systemd/system/snap.microk8s.daemon-containerd.service.d/resources.conf
    sudo systemctl daemon-reload
    sudo snap restart microk8s

# Check both mounts, the API encryption key, and node status.
verify:
    findmnt /var/lib/monadiam/microk8s/volumes
    findmnt /var/lib/monadiam/zot
    sudo test -s /etc/monadiam/microk8s/encryption.key
    sudo microk8s kubectl get nodes -o wide

# Start the separately installed Redis backup service.
redis-backup:
    sudo systemctl start monadiam-redis-backup@{{ profile }}.service

# Show the Redis backup timer and the service's last result.
redis-backup-status:
    systemctl status monadiam-redis-backup@{{ profile }}.timer --no-pager
    systemctl show monadiam-redis-backup@{{ profile }}.service -p Result -p ExecMainStatus -p MemoryPeak

# Follow the Redis backup service logs.
redis-backup-logs:
    journalctl -u monadiam-redis-backup@{{ profile }}.service -f
