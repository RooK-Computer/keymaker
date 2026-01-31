# Keymaker

Keymaker helps you manage your cartridges contents, and all you need is a capable RooK.

It allows you to:

* Copy games onto retropie setups on cartridges
* Copy games from retropie setups on cartridges
* write complete images onto cartridges

and all you need is your RooK and a device with a web browser.

## compatible RooKs

* RooK Mk3.1 + Lifeline Mod

## Components

Keymaker consists of a few moving parts:

* Shell scripts wrap the operating system operations and need to be put in $PATH.
* The core of Keymaker - an application in go, which shows some fancy UI on the Screen connected to RooK
* An HTTP API to do all the management.
* A Web App that interacts with said API

## Development (simulator + web UI)

You can run the API simulator on your dev machine and point the Vite dev server at it.

One terminal (API simulator):

`make sim-dev`

Another terminal (web UI):

`make web-dev`

Or run both together:

`make dev`

Notes:

* `sim-dev` runs the simulator with `--dev`, which enables permissive CORS for local development.
* The web UI reads the API host from `VITE_API_BASE_URL` (default: `http://127.0.0.1:8080`).
* Override the simulator port with `SIM_PORT=8090 make dev`.

## API validation

Run a lightweight simulator check against the OpenAPI expectations:

`make validate-sim-api`

Note: validation helpers live under `tools/` (the `scripts/` folder is reserved for scripts that get copied onto the device).

