BUILDTYPE  ?= debug
BUILD_DIR  ?= _build-$(BUILDTYPE)
PREFIX     ?= /usr/local
MESON_OPTS ?=

# pkg-config module:min-version pairs (version may be empty for "any")
PC_DEPS := gio-unix-2.0:2.80 gtk4:4.14 libadwaita-1:1.6 json-glib-1.0:1.6 vte-2.91-gtk4:0.79 libportal-gtk4:

TOOLS := meson ninja cc

.PHONY: check-deps setup configure build install uninstall clean

all: build

check-deps:
	@fail=0; \
	for t in $(TOOLS); do \
		cmd=$$t; \
		[ "$$t" = "cc" ] && { command -v cc >/dev/null 2>&1 || cmd=gcc; }; \
		if command -v "$$cmd" >/dev/null 2>&1; then \
			printf '  \033[1;32mok\033[0m    %-20s %s\n' "$$t" "$$(command -v "$$cmd")"; \
		else \
			printf '  \033[1;31mmissing\033[0m %-20s\n' "$$t"; \
			fail=1; \
		fi; \
	done; \
	for dep in $(PC_DEPS); do \
		name=$${dep%%:*}; min=$${dep#*:}; \
		if pkg-config --exists "$$name"; then \
			inst=$$(pkg-config --modversion "$$name"); \
			if [ -n "$$min" ] && ! pkg-config --atleast-version="$$min" "$$name"; then \
				printf '  \033[1;31mtoo old\033[0m %-20s %s (need >= %s)\n' "$$name" "$$inst" "$$min"; \
				fail=1; \
			else \
				printf '  \033[1;32mok\033[0m    %-20s %s%s\n' "$$name" "$$inst" "$${min:+ (need >= $$min)}"; \
			fi; \
		else \
			printf '  \033[1;31mmissing\033[0m %-20s%s\n' "$$name" "$${min:+ (need >= $$min)}"; \
			fail=1; \
		fi; \
	done; \
	if [ $$fail -ne 0 ]; then \
		echo; \
		echo "Some dependencies are missing or outdated."; \
		echo "Install the corresponding development packages for your distro,"; \
		echo "then re-run 'make check-deps'."; \
		exit 1; \
	fi; \
	echo; \
	echo "All dependencies satisfied."

# BUILDTYPE selects both the meson buildtype and the build directory
# (BUILD_DIR defaults to _build-$(BUILDTYPE)), so debug and release stay
# in separate directories and can be switched between without reconfiguring:
#   make build                  # debug, in _build-debug/
#   make build BUILDTYPE=release  # release, in _build-release/
#
# check-deps is intentionally not a prerequisite of any target below; it
# only runs when invoked explicitly with 'make check-deps'.

setup:
	@if [ -f "$(BUILD_DIR)/build.ninja" ]; then \
		echo "Build directory already configured at $(BUILD_DIR)."; \
		echo "Run 'make configure' to change options."; \
	else \
		echo "Configuring with meson setup ..."; \
		meson setup "$(BUILD_DIR)" --prefix=$(PREFIX) --buildtype=$(BUILDTYPE) $(MESON_OPTS); \
	fi

configure:
	@if [ ! -f "$(BUILD_DIR)/build.ninja" ]; then \
		$(MAKE) --no-print-directory setup; \
	fi
	@if [ -n "$(MESON_OPTS)" ]; then \
		echo "Reconfiguring with meson configure $(MESON_OPTS) ..."; \
		meson configure "$(BUILD_DIR)" $(MESON_OPTS); \
	else \
		echo "Current meson configuration:"; \
		meson configure "$(BUILD_DIR)"; \
	fi

build:
	@if [ ! -f "$(BUILD_DIR)/build.ninja" ]; then \
		$(MAKE) --no-print-directory setup; \
	fi
	@echo "Compiling with meson compile ..."
	meson compile -C "$(BUILD_DIR)"

install:
	@if [ ! -f "$(BUILD_DIR)/build.ninja" ]; then \
		echo "Run 'make build' first."; \
		exit 1; \
	fi
	@echo "Installing with meson install ..."
	sudo meson install -C "$(BUILD_DIR)"

uninstall:
	@if [ ! -f "$(BUILD_DIR)/build.ninja" ]; then \
		echo "Not configured; nothing to uninstall."; \
		exit 1; \
	fi
	@echo "Uninstalling with ninja uninstall ..."
	sudo ninja -C "$(BUILD_DIR)" uninstall

clean:
	@rm -rf _build-*
	@echo "Removed all _build-* directories"