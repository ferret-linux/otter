"""Shared model layer for otter completion generation.

Parses internal/cli/show-file/*.help into a Command tree and exposes the
hand-maintained flag-type/special-value tables used by gen_bash.py,
gen_zsh.py, and gen_fish.py. Not runnable on its own -- see those scripts
(or `make completions`) to actually generate completions.
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