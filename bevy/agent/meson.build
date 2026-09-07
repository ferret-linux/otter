# We build the agent with older versions of GLib required as well
# as various linker symbols so that we can run the agent, unmodified
# on older enterprise linux distributions. Currently, bevy-agent
# can run on CentOS 7.

BEVY_AGENT_MIN_GLIB_MAJOR = 2
BEVY_AGENT_MIN_GLIB_MINOR = 56
BEVY_AGENT_MIN_GLIB_VERSION = '@0@.@1@'.format(BEVY_AGENT_MIN_GLIB_MAJOR, BEVY_AGENT_MIN_GLIB_MINOR)

bevy_agent_deps = [
  dependency('gio-2.0', version: '>= @0@'.format(BEVY_AGENT_MIN_GLIB_VERSION)),
  dependency('gio-unix-2.0', version: '>= @0@'.format(BEVY_AGENT_MIN_GLIB_VERSION)),
  dependency('json-glib-1.0'),
]

bevy_agent_ipc = gnome.gdbus_codegen('bevy-agent-ipc',
           sources: 'dev.itznoel.Bevy.Agent.xml',
  interface_prefix: 'dev.itznoel.Bevy.',
         namespace: 'BevyIpc',
)

bevy_agent_sources = [
  bevy_agent_ipc,
  files([
    'bevy-agent-util.c',
    'bevy-container-provider.c',
    'bevy-distrobox-container.c',
    'bevy-podman-container.c',
    'bevy-podman-provider.c',
    'bevy-process-impl.c',
    'bevy-run-context.c',
    'bevy-session-container.c',
    'bevy-toolbox-container.c',
    'bevy-unix-fd-map.c',
    'bevy-agent.c',
    'bevy-agent-impl.c',
  ]),
]

bevy_agent_link_args = []

bevy_agent_c_args = [
  release_link_args,
  '-DGLIB_VERSION_MIN_REQUIRED=GLIB_VERSION_@0@_@1@'.format(BEVY_AGENT_MIN_GLIB_MAJOR, BEVY_AGENT_MIN_GLIB_MINOR),
  '-DGLIB_VERSION_MAX_ALLOWED=GLIB_VERSION_@0@_@1@'.format(BEVY_AGENT_MIN_GLIB_MAJOR, BEVY_AGENT_MIN_GLIB_MINOR),
]

if get_option('libc-compat')
libc_compat = true
if target_machine.cpu_family() == 'x86_64'
  bevy_agent_c_args += ['-include', 'x86_64/force_link_glibc_2.17.h']
else
  libc_compat = false
endif

if libc_compat
  config_h.set10('LIBC_COMPAT', libc_compat)
  bevy_agent_link_args += ['-Wl,--wrap=__libc_start_main']
endif
endif

executable('bevy-agent', bevy_agent_sources,
         dependencies: bevy_agent_deps,
              install: true,
          install_dir: get_option('libexecdir'),
               c_args: bevy_agent_c_args,
            link_args: bevy_agent_link_args,
  include_directories: include_directories('..'),
)
