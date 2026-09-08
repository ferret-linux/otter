test_env = [
  'G_TEST_SRCDIR=@0@'.format(meson.current_source_dir()),
  'G_TEST_BUILDDIR=@0@'.format(meson.current_build_dir()),
  'G_DEBUG=gc-friendly',
  'GSETTINGS_BACKEND=memory',
  'MALLOC_CHECK_=2',
]

testsuite_c_args = [
  '-DG_LOG_DOMAIN="bevy"',
  '-DDISABLE_AGENT_ENTRY_POINT',
  '-DG_ENABLE_DEBUG',
  '-UG_DISABLE_ASSERT',
  '-UG_DISABLE_CAST_CHECKS',
]

testsuite = {
  'test-podman' : {},
  'test-palette-search' : {
    'extra_sources' : [
      '../src/bevy-palette-search.c',
    ],
  },
}

testsuite_deps = [
  bevy_agent_deps,
]

foreach test, params: testsuite
  test_exe = executable(test,
    ['@0@.c'.format(test)] + params.get('extra_sources', []) + bevy_agent_sources,
                 c_args: testsuite_c_args,
           dependencies: testsuite_deps,
    include_directories: [include_directories('..'), include_directories('../src'), include_directories('.'), include_directories('../agent')],
  )
  if not params.get('skip', false)
    test(test, test_exe, env: test_env)
  endif
endforeach
