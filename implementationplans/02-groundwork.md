# Rook Cartridge Writer Assistant, Implementation Plan 1: scripts

The software we're going to build here will be run on a custom retro gaming console which utilizes cartridges.
It is built on Raspberry PI CM4 Modules and the cartridges are essentially SD Card slots. 
It will be put into a cartridge which copies itself into RAM. Afterwards the user can install a cartridge he wants to change the contents of.
More details will be added lateron.

This implementation plan has a heritage; all prior ones are considered common knowldege.

It's time to build the foundation for our application. To do that we have to learn what the core functionality of it will be. Once the system has booted from ramdisk, control is finally handed over to our tool, which will present a graphical UI in the style of a retro console; it has an 8-bit-like pixelated look. Immediately after starting, it has to tell the user to remove the cartridge.
Once the user has removed the cartridge, it has to tell the user to insert the cartridge to be written to into the rook.
After that, the regular UI will be shown, which shows WIFI credentials (including an QR code) and the IP Address of the website integrated into the application as well as the functions of the three buttons present on RooK. The website will be the interface the user will use to upload content onto the cartridges.

To built this, a few things need to be known about the environment: the application has to be able to execute the scripts already present using sudo. the linux environment does not come with an X server, so direct framebuffer interaction is the way to go. The three physical buttons are connected to GPIO pins.

Everything needs a name, so this tool will be called: Keymaker.

A different LLM has prepared a some more details and made a few design decisions to be adhered to:

## Purpose

This document defines the technical implementation plan for a cartridge flashing tool running on Raspberry Pi Compute Module 4 and 5. The system operates without X11/Wayland, uses the Linux framebuffer for local output, and exposes a browser-based UI for all user interaction.

The document is intended to be iteratively refined.

---

## 1. Goals and Non‑Goals

### Goals

* Run on Debian Trixie Lite from RAM
* No graphical stack (no X11 / Wayland)
* Minimal local status display (informational only)
* Browser-based UI for uploads and control
* Streaming flash pipeline without intermediate storage
* Reuse existing sudo-based shell scripts

### Non‑Goals

* Full local UI or file browser
* Complex animations or GPU acceleration
* On-device game launching

---

## 2. System Overview

The system consists of a single Go binary with three logical subsystems:

* Framebuffer status renderer
* HTTP server (primary UI)
* Flash orchestration and hardware control

All subsystems share a single immutable state snapshot updated by the flash controller.

---

## 3. Technology Stack

### Programming Language

* Go (single static binary preferred)

### Local Display

* Linux framebuffer (/dev/fb0)
* Software rendering only

### HTTP Interface

* Go net/http standard library

### Privileged Operations

* Existing shell scripts executed via sudo

---

## 4. Libraries

### Framebuffer

* github.com/gonutz/framebuffer

Used to map /dev/fb0 into a draw.Image-compatible surface.

### Text Rendering

* golang.org/x/image/font
* golang.org/x/image/font/opentype
* golang.org/x/image/draw

Used to render TTF fonts directly into an offscreen image buffer.

### QR Codes

* github.com/skip2/go-qrcode

Used for Wi-Fi and HTTP URL QR code generation.

### GPIO (Optional)

Only required if local buttons are enabled later.

* github.com/warthog618/gpiod
  or
* periph.io/x/conn/v3

---

## 5. Runtime Architecture

### Process Model

* Single Go process
* No background daemons
* No IPC between binaries

### Concurrency

* Renderer loop runs continuously
* HTTP handlers are event-driven
* Flash job runs in its own goroutine
* Shared state protected via mutex or atomic value

---

## 6. State Model

### States

* BOOTING
* READY
* FLASHING
* DONE
* ERROR
* CANCELLED

### Shared State Fields

* Wi-Fi

  * SSID
  * Password
  * Wi-Fi QR payload

* Network

  * IP address
  * HTTP URL
  * URL QR payload

* Flash

  * Target block device
  * Bytes written
  * Write rate (optional)
  * Status message
  * Error message

The renderer and HTTP status endpoint consume read-only snapshots of this state.

---

## 7. Block Device Detection

Cartridges appear as Linux block devices connected via the SD interface.

### Detection Strategy

* Prefer stable paths under /dev/disk/by-path or /dev/disk/by-id
* If needed, delegate detection to an existing shell script

### Validation

* Path exists
* Device is a block device
* Device is writable

The resolved device path is stored as the flash target.

---

## 8. Network and Wi‑Fi Handling

### Setup

* Wi-Fi configuration is performed via an existing sudo shell script
* Tool assumes Wi-Fi is ready before entering READY state

### IP Detection

* IP address is obtained via a shell command
* Parsed and stored in shared state

### QR Codes

Two QR codes are generated:

* Wi-Fi credentials (standard Wi-Fi QR format)
* HTTP URL pointing to the device

---

## 9. HTTP API

### Endpoints

* GET /

  * Minimal HTML UI
  * Upload form
  * Flash and cancel controls

* GET /status

  * JSON representation of shared state

* POST /flash

  * Accepts img.gz upload
  * Starts streaming flash job

* POST /cancel

  * Cancels the active flash job

### Upload Rules

* Request body is streamed
* No full buffering in memory or disk
* Best-effort validation of file type

---

## 10. Flash Pipeline

### Requirements

* Input format: gzip-compressed raw disk image
* No temporary storage
* Direct write to block device

### Execution Model

* HTTP request body is passed directly to gunzip
* gunzip output is piped directly into dd
* dd writes to the target block device

### Privileges

* dd and related scripts are executed via sudo
* Go process itself remains unprivileged

---

## 11. Progress Reporting

### Source

* dd progress output (stderr)

### Handling

* stderr is parsed line by line
* Bytes written are extracted
* Shared state is updated accordingly

### Display

* Browser UI polls /status
* Local framebuffer interpolates progress for smooth animation

Percent completion is optional and only shown if reliable size information is available.

---

## 12. Cancellation

### Mechanism

* Flash processes are started in their own process group
* Cancellation sends SIGTERM to the group
* SIGKILL is used if termination does not complete

### State Transition

* FLASHING → CANCELLED → READY

---

## 13. Framebuffer Rendering

### Rendering Strategy

* Offscreen logical canvas (e.g. 320x240)
* Nearest-neighbor scaling to framebuffer resolution
* Fixed layout:

  * Header
  * Status text
  * QR codes
  * Progress bar

### Fonts

* One bundled monospace or retro-style TTF
* Multiple sizes loaded at startup

---

## 14. Sudo and Security Model

### Principles

* Minimize privileged surface area
* Prefer wrapper scripts over direct binary execution
* The wrapper scripts in the scripts subfolder are placed in /usr/bin and therefore available via the $PATH variable.
* Use absolute paths only

### Sudoers

* Allow only specific commands
* No shell expansion or wildcards

---

## 15. Packaging and Deployment

* Single Go binary
* Embedded assets:

  * HTML UI
  * Fonts

### Runtime Dependencies

* sudo
* gzip
* coreutils
* optional: pv

---

## 16. Testing Scope

* Framebuffer rendering on CM4 and CM5
* Block device detection with real cartridges
* Flash success and cancellation
* Upload stability over Wi-Fi
* Concurrent HTTP status polling

Power-loss behavior during flashing must be documented explicitly.


## Implementation steps


However this implementation plan defines how to lay the foundation. We're only shortly tackling specific features.
Here is a step-by-step guide on how to proceed:

### Step one: prepare the environment

Currently, here is no go code in this project. Create all the nessecary files so that we can compile the application. Build a main function, create the needed go-description files, prepare a test compile. To ease development, also create a Makefile which in turn calls go to build it.

### Step two: architecture

Once that's done, we're going to build the basic architecture. There are a few things to consider:

* We're going to need images which will be embedded into the binary
* The same for fonts.
* Lateron the HTML files for the website will also be embedded into the binary.

And there are multiple screens the application will show:

* the initial screen to tell the user to remove the cartridge
* the screen to tell the user to insert the cartridge to be modified
* the main screen with all the details discussed earlier

And of course there are a few components which will be used lateron: a component to manage the three physical buttons, a component to interact with the system (executing scripts, configuring WIFI, ...), a component for the web server just to name a few we will be implementing in the future.

None of that will be built in this step. This step's purpose is to build the code infrastructure (folders for things, basic interfaces, calling paths,...). So this is what we do here.

### Step three: implement the first screen

As a first MVP we're going to implement the first screen. The Backgroundcolour should be the same across all screens and should be easily changeable (for example by using a constant). It should show the rook logo (supplied by me) and a note "please remove the cartridge now". The text color should also be configured globally. 
Once the cartridge has been removed, the program should exit for now.