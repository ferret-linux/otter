bevy_sources = [
  'gdkhsla.c',
  'main.c',
  'bevy-add-button-list-item.c',
  'bevy-add-button-list-model.c',
  'bevy-application.c',
  'bevy-client.c',
  'bevy-close-dialog.c',
  'bevy-custom-link-editor.c',
  'bevy-custom-link-row.c',
  'bevy-custom-link.c',
  'bevy-find-bar.c',
  'bevy-fullscreen-box.c',
  'bevy-gated-list-model.c',
  'bevy-inspector.c',
  'bevy-menu-row.c',
  'bevy-palette.c',
  'bevy-palette-preview.c',
  'bevy-palette-preview-color.c',
  'bevy-palette-search.c',
  'bevy-parking-lot.c',
  'bevy-preferences-list-item.c',
  'bevy-preferences-window.c',
  'bevy-profile-editor.c',
  'bevy-profile-row.c',
  'bevy-profile.c',
  'bevy-session.c',
  'bevy-settings.c',
  'bevy-shortcut-accel-dialog.c',
  'bevy-shortcut-row.c',
  'bevy-shortcuts.c',
  'bevy-shrinker.c',
  'bevy-tab.c',
  'bevy-tab-monitor.c',
  'bevy-terminal.c',
  'bevy-theme-selector.c',
  'bevy-title-dialog.c',
  'bevy-profile-dialog.c',
  'bevy-user-palettes.c',
  'bevy-util.c',
  'bevy-window-dressing.c',
  'bevy-window.c',
  'eggshell.c',
]

bevy_enum_headers = [
  'bevy-profile.h',
  'bevy-settings.h',
  'bevy-tab.h',
  'bevy-tab-monitor.h',
]

bevy_sources += gnome.mkenums_simple('bevy-enums',
            sources: bevy_enum_headers,
  identifier_prefix: 'Bevy',
      symbol_prefix: 'bevy',
)

bevy_sources += gnome.compile_resources('bevy-resources',
  'bevy.gresource.xml',
  c_name: 'bevy'
)

bevy_sources += vcs_tag(
      command: ['git', 'describe'],
     fallback: meson.project_version(),
        input: 'bevy-build-ident.h.in',
       output: 'bevy-build-ident.h',
)

schema_config = configuration_data()
schema_config.set('APP_SCHEMA_ID', app_schema_id)
schema_config.set('APP_SCHEMA_PATH', app_schema_path)
schema_config.set('APP_SCHEMA_PROFILE_ID', app_schema_profile_id)
schema_config.set('APP_SCHEMA_SHORTCUTS_ID', app_schema_shortcuts_id)
schema_config.set('APP_SCHEMA_SHORTCUTS_PATH', app_schema_shortcuts_path)
configure_file(
          input: 'dev.itznoel.Bevy.gschema.xml.in',
         output: '@0@.gschema.xml'.format(app_schema_id),
  configuration: schema_config,
    install_dir: get_option('datadir') / 'glib-2.0' / 'schemas',
)

# VTE comes exclusively from the local vte-nf subproject (subprojects/vte,
# a symlink to the sibling otter/vte-nf tree). No system or upstream VTE is
# ever used. Only the gtk4 library is built; everything legacy or
# unrelated (gtk3, gir, docs, vapi, glade, app, sixel, icu, gnutls,
# terminfo, dbg, systemd scopes) is disabled; a11y and bidi stay on.
vte_local = subproject('vte',
                       version: vte_req,
                       default_options: ['gtk3=false',
                                         'gtk4=true',
                                         'gir=false',
                                         'docs=false',
                                         'vapi=false',
                                         'glade=false',
                                         'app=false',
                                         'sixel=false',
                                         'icu=false',
                                         'gnutls=false',
                                         'terminfo=false',
                                         'dbg=false',
                                         '_systemd=false'])
vte_dep = vte_local.get_variable('libvte_gtk4_dep')

bevy_deps = [
  dependency('gio-unix-2.0', version: glib_req),
  dependency('gtk4', version: gtk_req),
  dependency('libadwaita-1', version: adwaita_req),
  dependency('json-glib-1.0', version: json_glib_req),
  vte_dep,
  cc.find_library('m', required: false),
]

if host_machine.system() == 'linux'
  bevy_deps += [dependency('libportal-gtk4')]
endif

bevy_c_args = [
  release_link_args,
  '-DG_LOG_DOMAIN="Bevy"',
  '-DGLIB_VERSION_MIN_REQUIRED=GLIB_VERSION_@0@'.format(glib_req_version.replace('.', '_')),
  '-DGLIB_VERSION_MAX_ALLOWED=GLIB_VERSION_@0@'.format(glib_req_version.replace('.', '_')),
  '-DGDK_VERSION_MIN_REQUIRED=GDK_VERSION_@0@'.format(gtk_req_version.replace('.', '_')),
  '-DGDK_VERSION_MAX_ALLOWED=GDK_VERSION_@0@'.format(gtk_req_version.replace('.', '_')),
  '-DADW_VERSION_MIN_REQUIRED=ADW_VERSION_@0@'.format(adwaita_req_version.replace('.', '_')),
  '-DADW_VERSION_MAX_ALLOWED=ADW_VERSION_@0@'.format(adwaita_req_version.replace('.', '_')),
]

bevy_include_directories = [
  include_directories('.'),
  include_directories('..'),
  include_directories('../agent'),
]

executable('bevy', bevy_sources + bevy_agent_ipc,
         dependencies: bevy_deps,
              install: true,
               c_args: bevy_c_args,
  include_directories: bevy_include_directories,
)
