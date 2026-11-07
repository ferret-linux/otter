# Holt

Holt is the Electron desktop interface that lives alongside the Otter CLI.
It provides screens for listing and controlling containers, creating a
container, browsing the image registry, changing app settings, and viewing
the CLI command log. Holt invokes the `otter` executable installed or built
on the same machine; it is a graphical frontend to the CLI rather than a
separate container manager.

Holt is currently an in-repository development app. Its package is marked
private and the repository does not define a release installer for it.

## Requirements

- Node.js and npm compatible with the versions in `package-lock.json`
- A built or installed `otter` executable available on `PATH`
- A supported container manager configured for Otter (Podman, Docker, or
  nerdctl)

## Run in development

From the repository root, build the CLI and install Holt's locked npm
dependencies:

```sh
cd otter
make build
cd ../holt
npm ci
```

Then start the development app:

```sh
PATH="../otter/bin:$PATH" npm run electron:dev
```

The `electron:dev` script starts Vite, watches the Electron TypeScript files,
and launches Electron after the local Vite server is ready. If Otter is
already installed and available on `PATH`, the extra `PATH=...` prefix is not
needed.

## Package scripts

Run these from `holt/`:

| Command | Purpose |
| --- | --- |
| `npm run dev` | Start the Vite renderer development server. |
| `npm run build` | Type-check the Electron main process and build the renderer. |
| `npm start` | Build the app, then launch it with Electron. |
| `npm run electron:dev` | Run the renderer and Electron app together with development reloads. |

The UI communicates with Electron's main process through a narrow preload
API. That process runs the `otter` CLI and relays command output and status to
the interface. For CLI behavior and built-in documentation, see
[`../otter/README.md`](../otter/README.md).
