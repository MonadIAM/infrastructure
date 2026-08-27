require_env() {
    for name in "$@"; do
        if [ -z "$(printenv "$name")" ]; then
            echo "Missing required env: $name" >&2
            exit 1
        fi
    done
}

init_env_file() {
    cd "$ROOT_DIR"

    if [ -f .env ]; then
        return
    fi

    cp .env.example .env
    chmod 600 .env
    echo "created .env"
}

load_env() {
    init_env_file >/dev/null
    set -a
    . "$ROOT_DIR/.env"
    set +a
}
