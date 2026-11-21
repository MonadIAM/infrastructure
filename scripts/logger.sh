#!/usr/bin/env bash

ESC=$(printf "\033")
GREEN="${ESC}[32m"
RED="${ESC}[31m"
YELLOW="${ESC}[33m"
CYAN="${ESC}[36m"
BG_GREEN="${ESC}[42m"
BG_RED="${ESC}[41m"
RESET="${ESC}[0m"

LOG_NESTED="${LOG_NESTED:-0}"
LOG_INDENT="    "

if [ -n "${EPOCHREALTIME:-}" ]; then
    LOG_CLOCK="bash"
elif command -v perl >/dev/null 2>&1; then
    LOG_CLOCK="perl"
else
    LOG_CLOCK="date"
fi

now_ms() {
    case "$LOG_CLOCK" in
        bash)
            echo $((${EPOCHREALTIME/[.,]/} / 1000)) ;;
        perl)
            perl -MTime::HiRes -e 'printf("%d\n", Time::HiRes::time() * 1000)' ;;
        *)
            echo $(($(date +%s) * 1000)) ;;
    esac
}

format_duration() {
    local milliseconds="$1"

    if [ "$milliseconds" -lt 1000 ]; then
        printf "%s ms" "$milliseconds"
    else
        printf "%s.%s s" "$((milliseconds / 1000))" "$(((milliseconds % 1000) / 100))"
    fi
}

log_context() {
    local label="$1" item

    shift
    printf "${YELLOW}%s${RESET}\n" "$label"

    for item in "$@"; do
        printf "${YELLOW}  %s${RESET}\n" "$item"
    done
}

log_step() {
    if [ "$LOG_NESTED" -eq 0 ]; then
        printf "\n"
    fi

    printf "${CYAN}%s${RESET}\n" "$1"
}

log_finding() {
    printf "%s${RED}%s${RESET}\n" "$LOG_INDENT" "$1"
}

log_output() {
    sed "s/^/$LOG_INDENT/" "$1"
}

log_raw() {
    cat "$1"
}

log_result() {
    local status="$1" color

    case "$status" in
        passed) color="$GREEN" ;;
        failed) color="$RED" ;;
        *) color="$YELLOW" ;;
    esac

    if [ "$#" -lt 2 ]; then
        printf "%s${color}%s${RESET}\n" "$LOG_INDENT" "$status"
    else
        printf "%s${color}%s (%s)${RESET}\n" "$LOG_INDENT" "$status" "$(format_duration "$2")"
    fi
}

log_verdict() {
    local status="$1" label="$2" text="$3" background color

    if [ "$LOG_NESTED" -gt 0 ]; then
        return 0
    fi

    if [ "$status" = "passed" ]; then
        background="$BG_GREEN"
        color="$GREEN"
    else
        background="$BG_RED"
        color="$RED"
    fi

    printf "\n${background}%s${RESET}${color} %s${RESET}\n" "$label" "$text"
}
