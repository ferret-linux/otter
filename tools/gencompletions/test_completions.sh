#!/usr/bin/env bash
# shellcheck shell=bash
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors
#
# Verifies the generated completions in completions/{otter.bash,otter.zsh,otter.fish}
# actually load and behave correctly, so broken completions never get committed.
#
# Tier 1 (blocking): syntax validity for all three shells, a zsh load test under
# a real compinit (regression guard for the "_arguments: can only be called
# from completion function" crash), and content smoke checks for bash/fish.
#
# Tier 2 (best-effort, non-blocking): a zsh interactive candidate-content check
# via zpty. zpty timing can be flaky in constrained/CI environments, so a
# failure here is reported but does not fail the run.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
COMPLETIONS_DIR="${REPO_ROOT}/completions"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RESET='\033[0m'

FAILURES=()
WARNINGS=()

pass() { printf '%b✔ %s%b\n' "${GREEN}" "${1}" "${RESET}"; }
fail() { printf '%b✘ %s%b\n' "${RED}" "${1}" "${RESET}"; FAILURES+=("${1}"); }
warn() { printf '%b⚠ %s%b\n' "${YELLOW}" "${1}" "${RESET}"; WARNINGS+=("${1}"); }

for f in "${COMPLETIONS_DIR}/otter.bash" "${COMPLETIONS_DIR}/otter.zsh" "${COMPLETIONS_DIR}/otter.fish"; do
    if [[ ! -f "${f}" ]]; then
        fail "missing completion file: ${f}"
    fi
done
if [[ ${#FAILURES[@]} -gt 0 ]]; then
    printf '%bfailed: %s%b\n' "${RED}" "${FAILURES[*]}" "${RESET}"
    exit 1
fi

echo "== tier 1: syntax checks =="

if command -v bash >/dev/null 2>&1; then
    if bash -n "${COMPLETIONS_DIR}/otter.bash" 2>/tmp/otter_bash_syntax.err; then
        pass "bash: otter.bash syntax valid"
    else
        fail "bash: otter.bash syntax invalid: $(cat /tmp/otter_bash_syntax.err)"
    fi
else
    warn "bash not found, skipping bash syntax check"
fi

if command -v zsh >/dev/null 2>&1; then
    if zsh -n "${COMPLETIONS_DIR}/otter.zsh" 2>/tmp/otter_zsh_syntax.err; then
        pass "zsh: otter.zsh syntax valid"
    else
        fail "zsh: otter.zsh syntax invalid: $(cat /tmp/otter_zsh_syntax.err)"
    fi
else
    warn "zsh not found, skipping zsh syntax check"
fi

if command -v fish >/dev/null 2>&1; then
    if fish -n "${COMPLETIONS_DIR}/otter.fish" 2>/tmp/otter_fish_syntax.err; then
        pass "fish: otter.fish syntax valid"
    else
        fail "fish: otter.fish syntax invalid: $(cat /tmp/otter_fish_syntax.err)"
    fi
else
    warn "fish not found, skipping fish syntax check"
fi

echo
echo "== tier 1: zsh load test (regression guard) =="

if command -v zsh >/dev/null 2>&1; then
    # A broken completion file (e.g. one that eagerly invokes _arguments outside
    # a real completion context) does not reliably yield a non-zero exit status
    # here -- the trailing case/dispatch statement can still "succeed" even
    # after _arguments has already failed and printed to stderr. So the load
    # test checks BOTH exit code and that stderr is empty.
    zsh_load_status=0
    zsh -c "autoload -Uz compinit; compinit -u -d /tmp/otter_test_zcompdump; source '${COMPLETIONS_DIR}/otter.zsh'" \
        >/tmp/otter_zsh_load.out 2>/tmp/otter_zsh_load.err || zsh_load_status=$?

    if [[ ${zsh_load_status} -eq 0 && ! -s /tmp/otter_zsh_load.err ]]; then
        pass "zsh: otter.zsh sources cleanly under compinit (no errors)"
    else
        fail "zsh: sourcing otter.zsh under compinit produced an error: $(cat /tmp/otter_zsh_load.err)"
    fi
else
    warn "zsh not found, skipping zsh load test"
fi

echo
echo "== tier 1: bash content smoke checks =="

if command -v bash >/dev/null 2>&1; then
    bash_completion_lib=""
    for candidate in /usr/share/bash-completion/bash_completion /etc/bash_completion; do
        if [[ -f "${candidate}" ]]; then
            bash_completion_lib="${candidate}"
            break
        fi
    done

    if [[ -z "${bash_completion_lib}" ]]; then
        warn "bash-completion package not found, skipping bash content checks"
    else
        bash_result=$(bash -c "
            source '${bash_completion_lib}'
            source '${COMPLETIONS_DIR}/otter.bash'
            COMP_WORDS=(otter create --)
            COMP_CWORD=2
            COMP_LINE='otter create --'
            COMP_POINT=\${#COMP_LINE}
            _otter
            printf '%s\n' \"\${COMPREPLY[@]}\"
        " 2>/tmp/otter_bash_content.err)

        if echo "${bash_result}" | grep -qx -- "--image" && echo "${bash_result}" | grep -qx -- "--help"; then
            pass "bash: 'otter create --' offers expected flags"
        else
            fail "bash: 'otter create --' did not offer expected flags: $(cat /tmp/otter_bash_content.err)"
        fi

        top_result=$(bash -c "
            source '${bash_completion_lib}'
            source '${COMPLETIONS_DIR}/otter.bash'
            COMP_WORDS=(otter '')
            COMP_CWORD=1
            COMP_LINE='otter '
            COMP_POINT=\${#COMP_LINE}
            _otter
            printf '%s\n' \"\${COMPREPLY[@]}\"
        " 2>/tmp/otter_bash_top.err)

        if echo "${top_result}" | grep -qx -- "create" && echo "${top_result}" | grep -qx -- "enter"; then
            pass "bash: top-level completion offers expected subcommands"
        else
            fail "bash: top-level completion missing expected subcommands: $(cat /tmp/otter_bash_top.err)"
        fi
    fi
else
    warn "bash not found, skipping bash content checks"
fi

echo
echo "== tier 1: fish content smoke checks =="

if command -v fish >/dev/null 2>&1; then
    fish_result=$(fish -c "source '${COMPLETIONS_DIR}/otter.fish'; complete -C'otter cre'" 2>/tmp/otter_fish_content.err)
    if echo "${fish_result}" | grep -q "^create"; then
        pass "fish: 'otter cre<TAB>' offers 'create'"
    else
        fail "fish: 'otter cre<TAB>' did not offer 'create': $(cat /tmp/otter_fish_content.err)"
    fi

    fish_top=$(fish -c "source '${COMPLETIONS_DIR}/otter.fish'; complete -C'otter '" 2>/tmp/otter_fish_top.err)
    if echo "${fish_top}" | grep -q "^enter" && echo "${fish_top}" | grep -q "^registry"; then
        pass "fish: top-level completion offers expected subcommands"
    else
        fail "fish: top-level completion missing expected subcommands: $(cat /tmp/otter_fish_top.err)"
    fi
else
    warn "fish not found, skipping fish content checks"
fi

echo
echo "== tier 2 (best-effort): zsh interactive candidate check =="

if command -v zsh >/dev/null 2>&1; then
    zsh_pty_script=$(cat <<'ZPTY_EOF'
zmodload zsh/zpty
zpty otter_t "zsh -f"
zpty -w otter_t $'autoload -Uz compinit && compinit -u -d /tmp/otter_test_zcompdump\n'
sleep 0.4
zpty -r -t 1 otter_t out
zpty -w otter_t "source '${1}'"$'\n'
sleep 0.3
zpty -r -t 1 otter_t out
zpty -w otter_t $'otter crea\t'
sleep 0.6
zpty -r -t 1 otter_t out
zpty -d otter_t 2>/dev/null
print -r -- "${out}"
ZPTY_EOF
)
    if zsh_out=$(timeout 10 zsh -c "${zsh_pty_script}" "${COMPLETIONS_DIR}/otter.zsh" 2>/tmp/otter_zsh_pty.err); then
        if echo "${zsh_out}" | grep -q "create"; then
            pass "zsh (best-effort): interactive Tab completion offered 'create'"
        else
            warn "zsh (best-effort): interactive check ran but did not clearly show 'create' -- inspect manually if concerned"
        fi
    else
        warn "zsh (best-effort): interactive pty check did not complete cleanly (known to be timing-sensitive), skipping"
    fi
else
    warn "zsh not found, skipping best-effort interactive check"
fi

echo
if [[ ${#WARNINGS[@]} -gt 0 ]]; then
    printf '%bwarnings: %s%b\n' "${YELLOW}" "${WARNINGS[*]}" "${RESET}"
fi

if [[ ${#FAILURES[@]} -eq 0 ]]; then
    printf '%ball blocking checks passed%b\n' "${GREEN}" "${RESET}"
    exit 0
else
    printf '%bfailed checks: %s%b\n' "${RED}" "${FAILURES[*]}" "${RESET}"
    exit 1
fi