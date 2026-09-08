BUILDDIR ?= build
JOBS ?=

MESON = meson
NINJA = ninja

MESON_OPTS = \
	-Dgtk3=false \
	-Dgir=false \
	-Dvapi=false \
	-Ddocs=false \
	-Dglade=false \
	-Dapp=false \
	-Dsixel=false \
	-Dicu=false

NINJA_FLAGS = -C $(BUILDDIR)
ifneq ($(strip $(JOBS)),)
NINJA_FLAGS += -j$(JOBS)
endif

.PHONY: all setup reconfigure test check clean distclean install uninstall dist help

all: setup
	$(NINJA) $(NINJA_FLAGS)

setup: $(BUILDDIR)/build.ninja

$(BUILDDIR)/build.ninja:
	$(MESON) setup $(BUILDDIR) $(MESON_OPTS)

reconfigure:
	$(MESON) setup --reconfigure $(BUILDDIR) $(MESON_OPTS)

test check: all
	$(MESON) test -C $(BUILDDIR)

clean:
	$(NINJA) $(NINJA_FLAGS) clean

distclean:
	rm -rf $(BUILDDIR)

install: all
	$(NINJA) $(NINJA_FLAGS) install

uninstall:
	$(NINJA) $(NINJA_FLAGS) uninstall

dist: all
	$(MESON) dist -C $(BUILDDIR) --include-subprojects --no-tests

help:
	@echo "make                 build (setup first if needed)"
	@echo "make setup           meson setup $(BUILDDIR) with lean GTK4-only options"
	@echo "make reconfigure     re-run meson setup with the same options"
	@echo "make test|check      run the test suite"
	@echo "make clean           ninja clean"
	@echo "make distclean       remove $(BUILDDIR)"
	@echo "make install         install to prefix"
	@echo "make uninstall       uninstall"
	@echo "make dist            build a source tarball"
	@echo "overrides: BUILDDIR=...  JOBS=..."