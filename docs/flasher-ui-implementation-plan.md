# Flasher Browser UI Implementation Plan

## Goal

Implement the browser UI as a hierarchical, column-based workspace based on the
`flasher 0.1` layout direction:

- default 3 columns: `cartCol`, `osCol`, `contCol`
- strict left-to-right hierarchy for MVP
- RetroPie content management in `contCol`
- intentional empty state for unsupported OS content

## Column Roles

### `cartCol`

- always visible
- shows cartridge status (present, mounted, busy, detected OS)
- includes core actions:
  - flash cartridge (`.img.gz`)
  - eject/rebottle cartridge

### `osCol`

- read-only detected OS context for MVP
- no manual OS switching/override
- reflects what the backend can currently identify from cartridge info

### `contCol`

- RetroPie cartridge:
  - list games
  - upload game
  - download game
  - delete game
- unknown/non-RetroPie cartridge:
  - intentionally empty capability state
  - clear explanation that filesystem/content tools are not available yet

## State and Hierarchy Rules

- strict dependency: upstream context determines downstream context
- cartridge context change resets `osCol` and `contCol`
- OS context change resets `contCol`
- no preservation logic in MVP

## Responsive Rules

- adaptive columns first (shrink to fit)
- if viewport is too narrow, show active/relevant columns only (focus mode)
- this is a hybrid of adaptive fit and progressive disclosure

## Validation Checklist

- `cartCol` always renders with status and actions
- `osCol` reflects detected OS in read-only mode
- `contCol` enables RetroPie management only for RetroPie
- `contCol` shows intentional empty state for unsupported OS
- hierarchy reset behavior is consistent with strict MVP rules
- narrow viewport switches into active-column focus behavior

## Out of Scope (MVP)

- manual OS override
- generic filesystem browser for unknown OS types
- advanced content features (search, bulk actions, rich filtering)
- hardware-specific runtime behavior beyond simulator-backed browser validation
