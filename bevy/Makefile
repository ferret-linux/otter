BUILD_DIR         ?= _build
BUILD_DIR_RELEASE ?= _build-release
PREFIX            ?= /usr/local
BUILDTYPE         ?= debug
MESON_OPTS        ?=

# pkg-config module:min-version pairs (version may be empty for "any").
# vte is deliberately absent: it comes from the local vte-nf subproject only.
PC_DEPS := gio-unix-2.0:2.80 gtk4:4.14 libadwaita-1:1.6 json-glib-1.0:1.6 libportal-gtk4:

TOOLS := meson ninja cc

.PHONY: check-deps setup configure build build-release install install-release uninstall uninstall-release clean

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

setup: | check-deps
	@if [ -f "$(BUILD_DIR)/build.ninja" ]; then \
		echo "Build directory already configured at $(BUILD_DIR)."; \
		echo "Run 'make configure' to change options."; \
	else \
		echo "Configuring with meson setup ..."; \
		meson setup "$(BUILD_DIR)" --prefix=$(PREFIX) --buildtype=$(BUILDTYPE) $(MESON_OPTS); \
	fi

configure: | check-deps
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

build: | check-deps
	@if [ ! -f "$(BUILD_DIR)/build.ninja" ]; then \
		$(MAKE) --no-print-directory setup; \
	fi
	@echo "Compiling with meson compile ..."
	meson compile -C "$(BUILD_DIR)"

build-release: | check-deps
	@if [ ! -f "$(BUILD_DIR_RELEASE)/build.ninja" ]; then \
		echo "Configuring release build with meson setup ..."; \
		meson setup "$(BUILD_DIR_RELEASE)" --prefix=$(PREFIX) --buildtype=release $(MESON_OPTS); \
	else \
		echo "Reconfiguring release build ..."; \
		meson configure "$(BUILD_DIR_RELEASE)" --buildtype=release $(MESON_OPTS); \
	fi
	@echo "Compiling release build with meson compile ..."
	meson compile -C "$(BUILD_DIR_RELEASE)"

install: | check-deps
	@if [ ! -f "$(BUILD_DIR)/build.ninja" ]; then \
		echo "Run 'make build' first."; \
		exit 1; \
	fi
	@echo "Installing with meson install ..."
	sudo meson install -C "$(BUILD_DIR)"

install-release: | check-deps
	@if [ ! -f "$(BUILD_DIR_RELEASE)/build.ninja" ]; then \
		echo "Run 'make build-release' first."; \
		exit 1; \
	fi
	@echo "Installing release build with meson install ..."
	sudo meson install -C "$(BUILD_DIR_RELEASE)"

uninstall:
	@if [ ! -f "$(BUILD_DIR)/build.ninja" ]; then \
		echo "Not configured; nothing to uninstall."; \
		exit 1; \
	fi
	@echo "Uninstalling with ninja uninstall ..."
	sudo ninja -C "$(BUILD_DIR)" uninstall

uninstall-release:
	@if [ ! -f "$(BUILD_DIR_RELEASE)/build.ninja" ]; then \
		echo "Release build dir not configured; nothing to uninstall."; \
		exit 1; \
	fi
	@echo "Uninstalling release build with ninja uninstall ..."
	sudo ninja -C "$(BUILD_DIR_RELEASE)" uninstall

clean:
	@rm -rf "$(BUILD_DIR)"
	@echo "Removed $(BUILD_DIR)"