# Flasher Concept and UI Spec Bootstrap

## Purpose of this Document

This document is a practical, implementation-level bootstrap for future AI agents and contributors working on the Flasher browser app. It describes:

- current product intent and user value
- supported use cases and user flows
- current interaction model and state rules
- current UI/spec decisions (visual + layout)
- code structure and key files
- constraints, known trade-offs, and safe extension points

Use this as the source of truth before making changes.

---

## Product Intent (Current State)

Flasher is a browser-based cartridge manager for RooK devices. It enables users to:

- inspect cartridge status
- flash full cartridge images
- browse/select emulator systems
- manage game files for selected systems (download, delete, upload/add)
- reveal systems with no games yet and start populating them

Core value: no separate desktop utility is required; cartridge operations are available through a focused web UI.

---

## Primary Use Cases

## 1) Cartridge Operator Flow

User inserts cartridge and needs immediate status + control:

- see cartridge presence/state (`present`, `mounted`, `busy`, OS type)
- flash cartridge image (`.img.gz`)
- rebottle/eject cartridge

## 2) RetroPie Game Management Flow

User wants to work on game content:

- choose emulator/system in `osCol` (e.g. `nes`, `snes`)
- see games in `contCol` scoped to that system
- download or delete existing games
- add/upload game to current system

## 3) Empty-System Expansion Flow

User wants to add games to systems that currently have none:

- use `Add Game System` in `osCol`
- reveal systems present on cartridge but with 0 games
- select one and add first game

---

## Interaction Model (Current, Required)

The UI uses a 3-column hierarchical model:

- `cartCol` (left): cartridge and global cartridge actions
- `osCol` (middle): system selection
- `contCol` (right): content for selected system

### Hierarchy Rules

- `contCol` is scoped to selected system from `osCol`
- changing system updates `contCol` content context
- unsupported/non-RetroPie cartridges do not expose content operations

### Mobile/Narrow Behavior

- On narrow widths, active-column navigation is used (`Cartridge`, `System`, `Games`)
- selecting a system in narrow mode can move focus to `contCol`

---

## Current UI and Visual System Specs

## Layout

- viewport-oriented app shell
- yellow full-page background
- three columns with fixed first two widths and flexible third:
  - `280px | 280px | 1fr`
- 8px grid rhythm (`1u = 8px`) for spacing, gutters, paddings, and most line-height cadence

## Typography

- Iosevka is bundled locally (no runtime web dependency)
- headings and retro-emphasis controls use Iosevka
- sizing currently tuned down from earlier oversized iteration

## Color/Style Language

- background: yellow retro field
- panels: light grey gradient
- accent: purple for key labels/titles
- primary actions: yellow retro buttons with offset shadow
- secondary row actions (`Download`, `Delete`): refined compact retro style, natural casing

## Borders and Surfaces

- column borders intentionally softened (no harsh heavy outlines)
- row states in `osCol` and `contCol` use consistent default fill
- selected `osCol` row uses white fill + clear inset emphasis

---

## Column-Level Behavior Details

## `cartCol`

Contains:

- title `Cartridge`
- cartridge illustration
- detected OS summary (`RetroPie` / unknown)
- status line (`Mounted`, `Busy`)
- primary actions:
  - `Flash Cartridge`
  - `Rebottle Cartridge`

Notes:

- actions are visible in natural flow (not forced off-screen)

## `osCol`

Contains:

- title `System`
- detected OS text
- list of systems that currently have games
- `Add Game System` button
- optional revealed list of systems with no games

Behavior:

- `Add Game System` is always present
- reveals empty systems list when available
- can be disabled if none available
- list glitch previously fixed by stabilizing system classification and preventing remount side-effects

## `contCol`

Contains:

- dynamic title: `<SELECTED_SYSTEM_UPPER> Games` (fallback `Games`)
- game rows for selected system
- per-row secondary actions:
  - `Download`
  - `Delete`
- tail action (after list): `Add Game`

Important:

- list-tail action remains after list content
- explicit spacing exists between list and tail action

---

## Data and API Behavior (Current)

Core API usage from webapp:

- `getCartridgeInfo()`
- `listRetroPieGames(system)`
- `uploadRetroPieGame(system, game, file)`
- `deleteRetroPieGame(system, game)`
- direct download URL via `apiV1Url(...)`
- cartridge actions via flash/eject endpoints

### System Bucketing Logic

To support `Add Game System`, systems are bucketed into:

- systems with games
- systems without games

Implementation detail:

- bucketing checks each system with `listRetroPieGames(system)`
- fallback on API error keeps prior classification to avoid flicker
- reclassification tied to systems-set change, not every poll tick

---

## Current Code Map (Important Files)

## App Shell and Top-Level State

- `webapp/src/App.tsx`
- `webapp/src/App.module.css`

## Column Components

- `webapp/src/views/columns/CartridgeColumn.tsx`
- `webapp/src/views/columns/OSColumn.tsx`
- `webapp/src/views/columns/ContentColumn.tsx`
- `webapp/src/views/columns/FlasherColumns.module.css`

## API Client

- `webapp/src/api/client.ts`
- `webapp/src/api/index.ts`

## Entry / Font Loading

- `webapp/src/main.tsx`

## Document Title

- `webapp/index.html`

---

## Non-Goals / Out of Scope (Still True)

- manual OS override flow
- generic filesystem browser for non-RetroPie OS
- advanced content tooling (bulk actions, rich filtering, etc.)
- hardware-runtime parity concerns beyond browser + API behavior

---

## Design/Implementation Guardrails for Future Agents

1. Preserve the 3-column interaction model unless explicitly changed by product decision.
2. Do not reintroduce stacked old-flow as default.
3. Keep 8px grid discipline (`1u`) for spacing rhythm.
4. Keep local font loading (no external runtime font dependency).
5. Keep primary vs secondary action hierarchy visually distinct.
6. Avoid reintroducing connector/linkage decorations between columns unless specifically requested.
7. Do not force content to bottom via spacer hacks; keep natural flow and visibility.
8. Avoid viewport clipping that hides actionable UI.
9. Keep button casing natural (no forced uppercase style policy).
10. Run typecheck/build after substantive UI changes.

---

## Known Pitfalls (Historical, Already Encountered)

- Over-constrained viewport layout caused column bottoms to clip.
- Hidden overflow at container/grid level caused controls to be cut off.
- Spacer/auto-margin patterns pushed tail actions below visible area.
- Frequent system re-bucketing caused `osCol` visual glitches and missing add controls.
- Inconsistent row fill contrast across columns reduced readability.

When changing layout/state code, verify these regressions do not return.

---

## Suggested Verification Checklist (After Future Changes)

1. Cartridge/status actions visible and usable in `cartCol`.
2. System selection updates `contCol` scope correctly.
3. `Add Game System` reveals empty systems when applicable.
4. `Add Game` remains at list tail in `contCol`.
5. No clipping/cut-off at bottom of columns on standard viewport.
6. Narrow mode still allows full task completion.
7. Visual hierarchy remains coherent (primary/secondary actions).
8. `npm run typecheck` passes.
9. `npm run build` passes.

---

## Optional Next Iteration Ideas

- Add compact status badges for cartridge metadata.
- Introduce subtle keyboard focus styling for accessibility.
- Add per-column max-content testing fixtures for overflow behavior.
- Define explicit CSS token file for color/spacing to centralize theming.

