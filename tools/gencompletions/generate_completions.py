#!/usr/bin/env python3
"""Generate bash/zsh/fish completions for otter from internal/cli/show-file/*.help.

Run via `make completions` from the repo root.
"""
import os
import re
import sys

REPO_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
HELP_DIR = os.path.join(REPO_ROOT, "internal", "cli", "show-file")
OUT_DIR = os.path.join(REPO_ROOT, "completions")

COLOR_TOKEN_RE = re.compile(r"\{\d\}|\{R\}")
BOX_CHARS = "│┆┬┴├┤╭╮╰╯─"
HEADER_RE = re.compile(r"^▸\s*([A-Za-z][A-Za-z /]*):$")
SUMMARY_RE = re.compile(r"⦿\s*(.+)")


class Flag:
    def __init__(self, long, short, desc):
        self.long = long
        self.short = short
        self.desc = desc


class Command:
    def __init__(self, name, path):
        self.name = name          # e.g. "list"
        self.path = path          # e.g. ("registry", "list")
        self.alias = None
        self.description = ""
        self.flags = []
        self.children = {}        # name -> Command

    @property
    def full_key(self):
        return " ".join(self.path)


def strip_row(line):
    """Strip color tokens and leading/trailing box-drawing chars/whitespace."""
    line = COLOR_TOKEN_RE.sub("", line)
    line = line.strip()
    line = line.strip(BOX_CHARS)
    line = line.strip()
    return line


def is_border_only(line):
    clean = COLOR_TOKEN_RE.sub("", line).strip()
    return clean != "" and all(c in BOX_CHARS or c.isspace() for c in clean)


def parse_help_file(filepath):
    """Returns (usage_lines: list[str], flags: list[Flag], description: str)."""
    with open(filepath, encoding="utf-8") as f:
        raw_lines = f.read().split("\n")

    section = None
    flags = []
    usage_lines = []
    description = ""

    for raw in raw_lines:
        clean_for_header = COLOR_TOKEN_RE.sub("", raw).strip()
        clean_for_header = clean_for_header.strip("│").strip()

        if not description:
            m_sum = SUMMARY_RE.search(clean_for_header)
            if m_sum:
                description = m_sum.group(1).strip()

        m = HEADER_RE.match(clean_for_header)
        if m:
            section = m.group(1).strip().lower()
            continue

        if is_border_only(raw) or not COLOR_TOKEN_RE.sub("", raw).strip():
            continue

        row = strip_row(raw)
        if not row:
            continue

        if section == "usage":
            usage_lines.append(row)
            continue

        if section not in ("options", "extras"):
            continue

        if "┆" in row:
            left, right = row.split("┆", 1)
            name = left.strip()
            right = right.strip()
            parts = right.split(None, 1)
            short = parts[0] if parts else None
            desc = parts[1] if len(parts) > 1 else ""
        else:
            parts = row.split(None, 1)
            if not parts:
                continue
            name = parts[0]
            short = None
            desc = parts[1] if len(parts) > 1 else ""

        if not name.startswith("--"):
            # not a flag row (shouldn't normally happen in Options/Extras)
            continue
        if name == "--":
            # positional separator, not a real flag
            continue

        flags.append(Flag(name, short, desc))

    return usage_lines, flags, description


def build_tree():
    root = Command("otter", ())

    filenames = sorted(f for f in os.listdir(HELP_DIR) if f.endswith(".help"))

    for fname in filenames:
        stem = fname[:-len(".help")]  # e.g. "otter_registry_list"
        if stem == "otter":
            _usage_lines, flags, description = parse_help_file(os.path.join(HELP_DIR, fname))
            root.flags = flags
            root.description = description
            continue

        assert stem.startswith("otter_"), f"unexpected help filename: {fname}"
        parts = tuple(stem[len("otter_"):].split("_"))

        node = root
        for depth, part in enumerate(parts):
            path_so_far = parts[: depth + 1]
            if part not in node.children:
                node.children[part] = Command(part, path_so_far)
            node = node.children[part]

        usage_lines, flags, description = parse_help_file(os.path.join(HELP_DIR, fname))
        node.flags = flags
        node.description = description

        # The word at index len(parts) in each usage line is the name/alias
        # for *this* command (e.g. "otter dmf mk ..." -> index 2 -> "mk").
        word_index = len(parts)
        canonical = node.name
        candidates = set()
        for line in usage_lines:
            words = line.split()
            if len(words) > word_index:
                candidates.add(words[word_index])
        candidates.discard(canonical)
        if len(candidates) == 1:
            node.alias = candidates.pop()
        elif len(candidates) > 1:
            print(
                f"warning: ambiguous alias candidates {candidates} for "
                f"{'/'.join(parts)}, leaving unset",
                file=sys.stderr,
            )

    return root


def walk(cmd):
    yield cmd
    for child in cmd.children.values():
        yield from walk(child)


# ---------------------------------------------------------------------------
# Supplement data: NOT derivable from the .help files themselves. Sourced by
# hand from internal/cli/*.go (cli.BoolFlag vs cli.StringFlag/StringSliceFlag/
# IntFlag) and from the previous hand-written completions' dynamic behaviour.
# If a command gains/loses a flag in Go without this table being updated, the
# generator will print a warning instead of silently guessing.
# ---------------------------------------------------------------------------

# (command path tuple) -> {flag long name (without "--"): "bool" | "value"}
FLAG_TYPES = {
    (): {"help": "bool", "version": "bool"},
    ("assemble",): {"file": "value", "replace": "bool", "help": "bool"},
    ("assemble", "create"): {"file": "value", "replace": "bool", "help": "bool"},
    ("assemble", "remove"): {"file": "value", "help": "bool"},
    ("create",): {
        "home": "value", "init": "bool", "root": "bool", "clone": "value",
        "image": "value", "shell": "value", "memory": "value", "nvidia": "bool",
        "volume": "value", "hostname": "value", "no-entry": "bool",
        "platform": "value", "init-hooks": "value", "cpu-threads": "value",
        "unshare-ipc": "bool", "unshare-all": "bool", "always-pull": "bool",
        "unshare-netns": "bool", "pre-init-hooks": "value", "unshare-groups": "bool",
        "unshare-devsys": "bool", "unshare-process": "bool", "no-userns-limit": "bool",
        "additional-flags": "value", "additional-packages": "value", "help": "bool",
        "disable-root-password-i-fully-understand-the-risks-and-accept-the-responsibilities": "bool",
    },
    ("enter",): {
        "root": "bool", "no-tty": "bool", "add-env": "value", "empty-env": "bool",
        "clean-path": "bool", "no-workdir": "bool", "auto-start": "bool",
        "additional-flags": "value", "help": "bool",
    },
    ("generate-entry",): {"all": "bool", "root": "bool", "icon": "value", "delete": "bool", "help": "bool"},
    ("inspect",): {"root": "bool", "json": "bool", "help": "bool"},
    ("journal",): {
        "tail": "value", "since": "value", "until": "value",
        "follow": "bool", "timestamps": "bool", "help": "bool",
    },
    ("list",): {"root": "bool", "json": "bool", "help": "bool"},
    ("lock",): {"all": "bool", "root": "bool", "help": "bool"},
    ("pause",): {"all": "bool", "root": "bool", "help": "bool"},
    ("registry",): {"help": "bool"},
    ("registry", "list"): {"all": "bool", "help": "bool"},
    ("registry", "pull"): {"all": "bool", "force": "bool", "help": "bool"},
    ("registry", "remove"): {"all": "bool", "force": "bool", "help": "bool"},
    ("remove",): {
        "all": "bool", "root": "bool", "force": "bool",
        "rm-home": "bool", "bypass-lock": "bool", "help": "bool",
    },
    ("restart",): {"all": "bool", "root": "bool", "force": "bool", "help": "bool"},
    ("start",): {"all": "bool", "root": "bool", "help": "bool"},
    ("stop",): {"all": "bool", "root": "bool", "force": "bool", "help": "bool"},
    ("unlock",): {"all": "bool", "root": "bool", "help": "bool"},
    ("upgrade",): {"all": "bool", "root": "bool", "running": "bool", "help": "bool"},
}

# (command path tuple, flag long name) -> ("enum", (values...)) | ("dynamic", "containers"|"registry_images")
SPECIAL_VALUES = {
    ("create",): {
        "image": ("dynamic", "registry_images"),
        "clone": ("dynamic", "containers"),
        "shell": ("enum", ("bash", "zsh", "fish")),
        "platform": ("enum", ("linux/amd64", "linux/arm64", "linux/arm/v7", "linux/386")),
    },
}

# command path tuple -> "containers" | "registry_images", for positional-arg completion
POSITIONAL_DYNAMIC = {
    ("enter",): "containers",
    ("remove",): "containers",
    ("start",): "containers",
    ("stop",): "containers",
    ("restart",): "containers",
    ("lock",): "containers",
    ("unlock",): "containers",
    ("pause",): "containers",
    ("inspect",): "containers",
    ("journal",): "containers",
    ("generate-entry",): "containers",
    ("upgrade",): "containers",
    ("registry", "pull"): "registry_images",
    ("registry", "remove"): "registry_images",
}


def flag_kind(path, flag_long):
    """Returns 'bool' or 'value' for a flag, warning + defaulting to 'bool' if unknown."""
    name = flag_long.lstrip("-")
    table = FLAG_TYPES.get(path, {})
    if name in table:
        return table[name]
    print(
        f"warning: no FLAG_TYPES entry for '{flag_long}' on '{' '.join(path) or '(root)'}' "
        f"-- defaulting to bool. Update FLAG_TYPES in this script if that's wrong.",
        file=sys.stderr,
    )
    return "bool"


def special_for(path, flag_long):
    name = flag_long.lstrip("-")
    return SPECIAL_VALUES.get(path, {}).get(name)


# ---------------------------------------------------------------------------
# Bash renderer
# ---------------------------------------------------------------------------

def _bash_value_candidates(kind_info):
    """kind_info is the ('enum', (...)) | ('dynamic', name) tuple from SPECIAL_VALUES."""
    sub_kind, payload = kind_info
    if sub_kind == "enum":
        words = " ".join(payload)
        return f'mapfile -t COMPREPLY < <(compgen -W "{words}" -- "${{cur}}")\n                    return\n                    ;;'
    # dynamic
    helper = "_otter_containers" if payload == "containers" else "_otter_registry_images"
    varname = "containers" if payload == "containers" else "images"
    return (
        f"local {varname}\n"
        f"                    {varname}=$({helper})\n"
        f'                    mapfile -t COMPREPLY < <(compgen -W "${{{varname}}}" -- "${{cur}}")\n'
        f"                    return\n"
        f"                    ;;"
    )


def render_bash(root):
    all_nodes = list(walk(root))[1:]  # skip root itself, it has no "path"

    # tree lookup table: "<parent path str>|<token>" -> "<resolved path str>"
    tree_entries = []
    for node in all_nodes:
        parent_path = " ".join(node.path[:-1])
        for token in [node.name] + ([node.alias] if node.alias else []):
            tree_entries.append(f'    ["{parent_path}|{token}"]="{node.full_key}"')

    lines = []
    lines.append("# bash completion for otter")
    lines.append("# Source this file or drop it in /etc/bash_completion.d/otter")
    lines.append("# or ~/.local/share/bash-completion/completions/otter")
    lines.append("#")
    lines.append("# Auto-generated by tools/gencompletions/generate_completions.py from")
    lines.append("# internal/cli/show-file/*.help -- do not edit by hand.")
    lines.append("# Run `make completions` to regenerate.")
    lines.append("")
    lines.append("_otter_containers() {")
    lines.append('    otter list --json 2>/dev/null | grep \'"name"\' | sed \'s/.*"name": *"\\([^"]*\\)".*/\\1/\'')
    lines.append("}")
    lines.append("")
    lines.append("_otter_registry_images() {")
    lines.append("    otter registry 2>/dev/null | awk 'NR>1 {print $1}'")
    lines.append("}")
    lines.append("")
    lines.append("_otter() {")
    lines.append("    local cur prev words cword")
    lines.append('    _init_completion -n "=:" || return')
    lines.append("")
    lines.append("    local -A _otter_tree=(")
    lines.extend(tree_entries)
    lines.append("    )")
    lines.append("")
    lines.append('    local path=""')
    lines.append("    local i")
    lines.append("    # shellcheck disable=SC2249")
    lines.append("    for (( i=1; i < cword; i++ )); do")
    lines.append('        local key="${path}|${words[i]}"')
    lines.append('        if [[ -n "${_otter_tree[$key]+x}" ]]; then')
    lines.append('            path="${_otter_tree[$key]}"')
    lines.append("        else")
    lines.append("            break")
    lines.append("        fi")
    lines.append("    done")
    lines.append("")
    lines.append('    case "${path}" in')
    lines.append('        "")')
    root_flag_names = " ".join(f.long for f in root.flags)
    top_commands = " ".join(n.name for n in all_nodes if len(n.path) == 1)
    lines.append(f'            mapfile -t COMPREPLY < <(compgen -W "{top_commands} {root_flag_names}" -- "${{cur}}")')
    lines.append("            ;;")

    for node in all_nodes:
        lines.append(f'        "{node.full_key}")')
        # prev-based special value completions
        special_cases = []
        plain_value_flags = []
        for f in node.flags:
            kind = flag_kind(node.path, f.long)
            if kind != "value":
                continue
            sp = special_for(node.path, f.long)
            if sp:
                special_cases.append(f"                {f.long})\n                    {_bash_value_candidates(sp)}")
            else:
                plain_value_flags.append(f.long)
        if plain_value_flags:
            joined = "|".join(plain_value_flags)
            special_cases.append(f"                {joined}) return ;;")
        if special_cases:
            lines.append("            # shellcheck disable=SC2249")
            lines.append('            case "${prev}" in')
            lines.extend(special_cases)
            lines.append("            esac")

        dynamic_target = POSITIONAL_DYNAMIC.get(node.path)
        if dynamic_target:
            helper = "_otter_containers" if dynamic_target == "containers" else "_otter_registry_images"
            varname = "containers" if dynamic_target == "containers" else "images"
            lines.append('            local has_positional=0')
            lines.append("            # shellcheck disable=SC2249")
            lines.append(f"            for (( i={len(node.path) + 1}; i < cword; i++ )); do")
            lines.append('                [[ "${words[i]}" != -* ]] && has_positional=1 && break')
            lines.append("            done")
            lines.append('            if [[ "${cur}" != -* && "${has_positional}" -eq 0 ]]; then')
            lines.append(f"                local {varname}")
            lines.append(f"                {varname}=$({helper})")
            lines.append(f'                mapfile -t COMPREPLY < <(compgen -W "${{{varname}}}" -- "${{cur}}")')
            lines.append("                return")
            lines.append("            fi")

        flag_names = " ".join(f.long for f in node.flags)
        child_names = " ".join(c.name for c in node.children.values())
        all_words = " ".join(w for w in [child_names, flag_names] if w)
        lines.append(f'            mapfile -t COMPREPLY < <(compgen -W "{all_words}" -- "${{cur}}")')
        lines.append("            ;;")

    lines.append("    esac")
    lines.append("}")
    lines.append("")
    lines.append("complete -F _otter otter")
    lines.append("")
    return "\n".join(lines)


# ---------------------------------------------------------------------------
# Zsh renderer
# ---------------------------------------------------------------------------

def _zsh_escape(desc):
    return desc.replace("'", "'\\''")


def _zsh_flag_spec(node, f):
    kind = flag_kind(node.path, f.long)
    desc = _zsh_escape(f.desc)
    if f.short:
        excl = f"'({f.long} {f.short})'{{{f.long},{f.short}}}"
    else:
        excl = f"'{f.long}'"

    if kind == "bool":
        return f"{excl}'[{desc}]'"

    sp = special_for(node.path, f.long)
    if sp is None:
        action = ""
    elif sp[0] == "enum":
        action = "(" + " ".join(sp[1]) + ")"
    elif sp[1] == "containers":
        action = "_otter_container_names"
    else:
        action = "_otter_registry_image_names"
    return f"{excl}'[{desc}]:value:{action}'"


def render_zsh(root):
    all_nodes = list(walk(root))

    lines = []
    lines.append("#compdef otter")
    lines.append("# zsh completion for otter")
    lines.append("# Drop in $fpath, e.g. /usr/local/share/zsh/site-functions/_otter")
    lines.append("# or ~/.local/share/zsh/site-functions/_otter (add dir to fpath first)")
    lines.append("#")
    lines.append("# Auto-generated by tools/gencompletions/generate_completions.py from")
    lines.append("# internal/cli/show-file/*.help -- do not edit by hand.")
    lines.append("# Run `make completions` to regenerate.")
    lines.append("")
    lines.append("_otter_container_names() {")
    lines.append("    local -a containers")
    lines.append('    containers=( ${(f)"$(otter list --json 2>/dev/null | grep \'"name"\' | sed \'s/.*"name": *"\\([^"]*\\)".*/\\1/\')"} )')
    lines.append("    compadd -a containers")
    lines.append("}")
    lines.append("")
    lines.append("_otter_registry_image_names() {")
    lines.append("    local -a images")
    lines.append("    images=( ${(f)\"$(otter registry 2>/dev/null | awk 'NR>1 {print $1}')\"} )")
    lines.append("    compadd -a images")
    lines.append("}")
    lines.append("")

    def emit_node(node):
        fn_suffix = "_".join(node.path).replace("-", "_")
        fn_name = f"_otter_{fn_suffix}" if node.path else "_otter"

        specs = [_zsh_flag_spec(node, f) for f in node.flags]
        positional = POSITIONAL_DYNAMIC.get(node.path)

        if node.children:
            lines.append(f"{fn_name}() {{")
            lines.append("    local state")
            lines.append("    _arguments -C \\")
            for spec in specs:
                lines.append(f"        {spec} \\")
            lines.append(f"        '1: :{fn_name}_subcommands' \\")
            lines.append("        '*:: :->sub' \\")
            lines.append("        && return")
            lines.append("")
            lines.append('    case "${state}" in')
            lines.append("        sub)")
            lines.append('            case "${words[1]}" in')
            for child in node.children.values():
                tokens = child.name + (("|" + child.alias) if child.alias else "")
                child_fn = f"_otter_{'_'.join(child.path).replace('-', '_')}"
                lines.append(f"                {tokens}) {child_fn} ;;")
            lines.append("            esac")
            lines.append("            ;;")
            lines.append("    esac")
            lines.append("}")
            lines.append("")

            lines.append(f"{fn_name}_subcommands() {{")
            lines.append("    local -a subcmds")
            lines.append("    subcmds=(")
            for child in node.children.values():
                desc = _zsh_escape(child.description)
                alias_note = f" (alias: {child.alias})" if child.alias else ""
                lines.append(f"        '{child.name}:{desc}{alias_note}'")
            lines.append("    )")
            label = node.name if node.path else "command"
            lines.append(f"    _describe '{label} subcommand' subcmds")
            lines.append("}")
            lines.append("")
        else:
            lines.append(f"{fn_name}() {{")
            lines.append("    _arguments \\")
            all_specs = list(specs)
            if positional == "containers":
                all_specs.insert(0, "'*:container:_otter_container_names'")
            elif positional == "registry_images":
                all_specs.insert(0, "'*:image:_otter_registry_image_names'")
            for idx, spec in enumerate(all_specs):
                sep = " \\" if idx < len(all_specs) - 1 else ""
                lines.append(f"        {spec}{sep}")
            lines.append("}")
            lines.append("")

        for child in node.children.values():
            emit_node(child)

    emit_node(root)

    lines.append("_otter \"$@\"")
    lines.append("")
    return "\n".join(lines)


# ---------------------------------------------------------------------------
# Fish renderer
# ---------------------------------------------------------------------------

def _fish_escape(desc):
    return desc.replace("'", "\\'")


def _fish_tokens(node):
    return [node.name] + ([node.alias] if node.alias else [])


def render_fish(root):
    all_nodes = list(walk(root))[1:]

    lines = []
    lines.append("# fish completion for otter")
    lines.append("# Drop in ~/.config/fish/completions/otter.fish")
    lines.append("#")
    lines.append("# Auto-generated by tools/gencompletions/generate_completions.py from")
    lines.append("# internal/cli/show-file/*.help -- do not edit by hand.")
    lines.append("# Run `make completions` to regenerate.")
    lines.append("")
    lines.append("complete -c otter -f")
    lines.append("")
    lines.append("function __otter_containers")
    lines.append('    otter list --json 2>/dev/null | string match -r \'"name": *"[^"]*"\' | string replace -r \'.*"name": *"([^"]*)".*\' \'$1\'')
    lines.append("end")
    lines.append("")
    lines.append("function __otter_registry_images")
    lines.append("    otter registry 2>/dev/null | tail -n +2 | awk '{print $1}'")
    lines.append("end")
    lines.append("")
    lines.append("function __otter_no_subcommand")
    lines.append("    not __fish_seen_subcommand_from \\")
    for node in [n for n in all_nodes if len(n.path) == 1]:
        toks = " ".join(_fish_tokens(node))
        lines.append(f"        {toks} \\")
    lines[-1] = lines[-1].rstrip(" \\")
    lines.append("end")
    lines.append("")
    lines.append("function __otter_using_subcommand")
    lines.append("    __fish_seen_subcommand_from $argv")
    lines.append("end")
    lines.append("")

    lines.append("# ── Global flags " + "─" * 62)
    for f in root.flags:
        short = f"-s {f.short.lstrip('-')} " if f.short else ""
        lines.append(f"complete -c otter -n __otter_no_subcommand -l {f.long.lstrip('-')} {short}-d '{_fish_escape(f.desc)}'")
    lines.append("")

    lines.append("# ── Top-level subcommands " + "─" * 54)
    for node in [n for n in all_nodes if len(n.path) == 1]:
        alias_note = f" (alias: {node.alias})" if node.alias else ""
        lines.append(
            f"complete -c otter -n __otter_no_subcommand -a {node.name} "
            f"-d '{_fish_escape(node.description)}{alias_note}'"
        )
    lines.append("")

    def ancestor_guard(node):
        """__otter_using_subcommand clauses for every ancestor level, ANDed together."""
        clauses = []
        cur = root
        for part in node.path[:-1]:
            cur = cur.children[part]
            toks = " ".join(_fish_tokens(cur))
            clauses.append(f"__otter_using_subcommand {toks}")
        return clauses

    def emit_node(node):
        toks = " ".join(_fish_tokens(node))
        own_clauses = ancestor_guard(node) + [f"__otter_using_subcommand {toks}"]

        if node.children:
            not_children = " ".join(t for c in node.children.values() for t in _fish_tokens(c))
            guard = "; and ".join(own_clauses) + f"; and not __fish_seen_subcommand_from {not_children}"
        else:
            guard = "; and ".join(own_clauses)

        lines.append(f"# ── {node.full_key} " + "─" * max(1, 70 - len(node.full_key)))
        for f in node.flags:
            short = f"-s {f.short.lstrip('-')} " if f.short else ""
            lines.append(f"complete -c otter -n '{guard}' -l {f.long.lstrip('-')} {short}-d '{_fish_escape(f.desc)}'")

        dynamic_target = POSITIONAL_DYNAMIC.get(node.path)
        if dynamic_target and not node.children:
            helper = "__otter_containers" if dynamic_target == "containers" else "__otter_registry_images"
            lines.append(f"complete -c otter -n '{guard}' -a '({helper})'")

        if node.children:
            for child in node.children.values():
                alias_note = f" (alias: {child.alias})" if child.alias else ""
                lines.append(
                    f"complete -c otter -n '{guard}' -a {child.name} "
                    f"-d '{_fish_escape(child.description)}{alias_note}'"
                )
        lines.append("")

        for child in node.children.values():
            emit_node(child)

    for node in [n for n in all_nodes if len(n.path) == 1]:
        emit_node(node)

    return "\n".join(lines)


if __name__ == "__main__":
    tree = build_tree()
    bash_out = render_bash(tree)
    zsh_out = render_zsh(tree)
    fish_out = render_fish(tree)
    os.makedirs(OUT_DIR, exist_ok=True)
    with open(os.path.join(OUT_DIR, "otter.bash"), "w") as f:
        f.write(bash_out)
    with open(os.path.join(OUT_DIR, "otter.zsh"), "w") as f:
        f.write(zsh_out)
    with open(os.path.join(OUT_DIR, "otter.fish"), "w") as f:
        f.write(fish_out)
    print("wrote bash + zsh + fish completions to", OUT_DIR, file=sys.stderr)