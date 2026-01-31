```markdown
# Rook Cartridge Writer Assistant, Implementation Plan 15: api simulator for development

The software we're going to build here will be run on a custom retro gaming console which utilizes cartridges.
It is built on Raspberry PI CM4 Modules and the cartridges are essentially SD Card slots.
It will be put into a cartridge which copies itself into RAM. Afterwards the user can install a cartridge he wants to change the contents of.
More details will be added lateron.

This implementation plan has a heritage; all prior ones are considered common knowldege.
It is important to keep one thing in mind when writing code: its not for the machines, its for humans. Keep that in mind, and thus refrain using single-character variable names. Instead come up with useful names for the variables.

## goal

The project has a set of OS-specific scripts, framebuffer rendering, and device-specific behavior.
For day-to-day development of the web ui and for testing workflows, we need a simulator that:

* runs on a regular development machine (linux/mac/windows)
* exposes the same http api as the real device (api/v1/...)
* returns useful default replies
* simulates the relevant workflows (insert cartridge, retropie cartridge, flash progress, eject, errors)
* does not use the linux framebuffer
* does not execute any shell scripts
* does not require privileged ports (port 80)

This plan implements that simulator as a separate executable, while reusing as much code as possible.

## decision: separate simulator entrypoint

We will implement the simulator as a second entrypoint in the same repository:

* `./` (root `main.go`) remains the "real" device application
* `./simulator` is the development simulator

Both will reuse the same api/router code from the existing `internal/web` package, but will use different implementations for the hardware/system dependencies.

This approach keeps a strong safety boundary (no accidental script/framebuffer access on dev machines) while avoiding duplication (both binaries share handlers, types, and state transitions).

## functional requirements

The simulator must:

* bind to an unprivileged port by default (e.g. 8080)
* allow overriding the listen address/port via flags and env vars
* support the web ui in two modes:
  * web ui served by vite dev server, talking to simulator api
  * (optional) serve embedded web ui assets (same as the device) for quick demos
* keep an in-memory or temp-dir backed "simulated cartridge" with a few scenarios:
  * no cartridge inserted
  * unknown cartridge
  * retropie cartridge with a few systems/games

The simulator should also support basic fault injection, at least for:

* flash failing mid-way
* mount failing
* cartridge eject failing

These can be controlled via flags at startup or via simulator-only endpoints.

## non-goals

* perfect emulation of all timing and hardware quirks
* implementing framebuffer drawing in a window
* simulating wifi hotspot behavior (unless needed by the web ui later)

## step one: define configuration for listen address and "mode"

Right now port 80 is not an option for most dev machines.
We need a consistent config model that both binaries can use.

* introduce a small config struct used by the http server startup:
  * listen address / port (default `:80` on device, default `:8080` in simulator)
  * a boolean to enable dev features (cors, more logging)
* allow overriding via:
  * flags (e.g. `--listen :8080`)
  * env vars (e.g. `KEYMAKER_LISTEN=:8080`)

If config is already present in the project, extend it instead of adding a second config mechanism.

## step two: separate the API wiring from the device startup

The existing code already has a web server and an api package.
We need a clean point where the api routes are registered without forcing framebuffer/system initialization.

* refactor the web server startup so that it takes dependencies as parameters (interfaces)
* ensure the route registration can be reused by both:
  * `./` (real)
  * `./simulator` (sim)

A good result of this step is that both entrypoints can build the http server like this:

* construct dependencies
* construct a web server
* register routes
* start listening

## step three: introduce interfaces for hardware/system dependencies

Handlers currently call into packages which may:

* execute scripts
* access the cartridge block device
* mount/unmount
* run flash
* interact with framebuffer / screens

We need to depend on interfaces, not concrete implementations.

Create interfaces for the minimal needs of the API layer, for example:

* cartridge state + metadata provider (cartridgeinfo)
* retropie filesystem access (list systems, list games, fetch game, delete game, upload game)
* flashing service (stream in, report progress)
* eject action (prepare for ejection)

The goal is to be able to swap out these dependencies without changing the HTTP handlers.

If interfaces already exist in the codebase, use them and expand them only when required.

## step four: implement simulator dependencies

Implement a simulator-backed set of dependencies.

Suggested approach:

* use a temp directory as "cartridge root", e.g. `/tmp/keymaker-sim/cartridge`
* provide seed data on startup (or on demand) to create:
  * `home/pi/RetroPie/roms/nes/...`
  * `home/pi/RetroPie/roms/snes/...`
* implement "flash" by consuming the incoming stream and simulating progress:
  * read bytes from request
  * increment a counter
  * periodically update a progress value
  * optionally discard data or write it to a temp file for debugging

The simulator should implement the same behavior constraints as the real server where it matters:

* do not buffer full uploads in memory
* stream request bodies

## step five: simulator-only control endpoints (optional but recommended)

To make the simulator useful for UI development, we need to be able to force the state.
We will add simulator-only endpoints under a separate path so the public api spec stays clean.

Example endpoints:

* [POST] /sim/reset
* [POST] /sim/scenario/{name}
* [POST] /sim/faults (enable/disable specific faults)

These endpoints are only registered by `./simulator`, never by `./`.

If we want to keep everything in OpenAPI docs, we can add a second OpenAPI file for simulator endpoints.

## step six: web ui development wiring

The simulator is primarily meant to support working on the web ui.

* add a documented way to run:
  * simulator on `localhost:8080`
  * vite dev server on `localhost:5173`
* configure the web ui to talk to the api base url via an env var (e.g. `VITE_API_BASE_URL`)

For local development, cors may be required; enable it in simulator mode.

## step seven: Makefile targets and documentation

Add make targets so developers do not need to remember the exact commands.

Suggested targets:

* `make sim` - runs the simulator binary (built into `bin/keymaker-sim`)
* `make web-dev` - runs the vite dev server
* `make dev` - optionally runs both in parallel (if that fits the current tooling)

Update the readme with:

* how to run the simulator
* how to run the web ui against it
* which scenarios exist
* how to change the port

## step eight: validate behavior against the API spec

The simulator must stay compatible with the real API.

* ensure routes and json structures match the existing OpenAPI specification
* add lightweight checks:
  * start simulator
  * curl a few endpoints
  * confirm they return valid json and expected status codes

If the project already has automated tests for handlers, add simulator coverage to those tests.

```