# Flasher Browser UI Update Plan: Cartridge Info API Shape Change

## Goal

Update the Flasher browser UI to consume the new `/cartridgeinfo` payload shape without changing the current 3-column interaction model.

The updated UI must:

- use `systems` from `/cartridgeinfo` as the source of systems that currently have games
- use `emptySystems` from `/cartridgeinfo` as the source of systems with no games yet
- keep `GET /retropie` behavior unchanged from the UI point of view
- preserve the current `cartCol` -> `osCol` -> `contCol` hierarchy
- remove the old client-side "bucket systems by probing every system" workaround

## Why This Change Exists

The backend now provides the system bucketing directly in `/cartridgeinfo`.

New behavior of `/cartridgeinfo`:

- `systems` is now an array of objects
- each entry contains:
  - `system`
  - `filecount`
- `emptySystems` is a separate string array
- `GET /retropie` still returns all known systems as a string array

This means the webapp no longer needs to infer "has games" vs "no games" by calling `listRetroPieGames(system)` for each system.

## Product / UX Intent

This update should preserve the current Flasher UX defined in `docs/webapp/flasher-concept-and-ui-spec-bootstrap.md` and `docs/webapp/flasher-ui-implementation-plan.md`.

Required behavior after the change:

- `osCol` lists systems with games directly from `/cartridgeinfo.systems`
- `Add Game System` reveals `emptySystems` directly from `/cartridgeinfo.emptySystems`
- selecting either kind of system still drives `contCol`
- `contCol` continues to load actual game names via `GET /retropie/{system}`
- non-RetroPie cartridges still show the intentional unsupported state
- narrow/mobile behavior remains unchanged

## Current Frontend Mismatch

The current frontend still assumes `info.systems` is `string[]`, which is visible in:

- `webapp/src/App.tsx`
- `webapp/src/views/columns/OSColumn.tsx`
- `webapp/src/views/columns/CartridgeColumn.tsx`
- `webapp/src/views/columns/ContentColumn.tsx`

The current `App.tsx` logic also performs extra bucketing by calling `listRetroPieGames(system)` for every detected system. That logic is now obsolete and should be removed.

## Data Model Update

Introduce the frontend assumption that cartridge info now has:

- `systems: Array<{ system: string; filecount: number }> | null`
- `emptySystems: string[] | null`

Derived UI data should become:

- `systemsWithGames: string[]`
- `emptySystems: string[]`
- optional display metadata for later use:
  - `filecount` per populated system

The UI should treat `/cartridgeinfo` as the source of truth for system bucketing.

## Implementation Steps

### 1. Update API-Driven Frontend Types

Adjust all frontend code that assumes `info.systems` is `string[]`.

Touch points likely include:

- `webapp/src/App.tsx`
- `webapp/src/views/columns/CartridgeColumn.tsx`
- `webapp/src/views/columns/ContentColumn.tsx`
- `webapp/src/views/columns/OSColumn.tsx`

Goal of this step:

- compile cleanly against the generated OpenAPI types
- stop treating `systems` as plain strings

### 2. Remove Client-Side System Bucketing Probes

Replace the old "probe every system with `listRetroPieGames(system)`" logic in `webapp/src/App.tsx`.

Current behavior to remove:

- build `withGames` and `withoutGames` by making one API request per system
- preserve prior classifications to avoid flicker

New behavior:

- derive `withGames` directly from `info.systems.map((entry) => entry.system)`
- derive `withoutGames` directly from `info.emptySystems ?? []`
- remove the loading state associated only with that bucketing workaround

This should reduce request volume and remove a source of UI instability already documented in the bootstrap doc.

### 3. Keep Selection and Hierarchy Rules Stable

Update the selected-system logic in `webapp/src/App.tsx` so it works with the new data shape while preserving current hierarchy behavior.

Rules should remain:

- cartridge context change resets downstream system/content context
- a valid selected system may come from either populated systems or empty systems
- preferred default selection should remain a system with games when available
- if there are no populated systems, fallback to the first empty system
- narrow-mode auto-navigation behavior should remain unchanged

### 4. Update `osCol` to Consume the New Data Shape

Keep the visual structure of `webapp/src/views/columns/OSColumn.tsx` intact, but feed it from backend-provided buckets.

Desired behavior:

- main system list shows populated systems
- `Add Game System` reveals empty systems
- button enable/disable behavior remains the same
- the component should not need to know how bucketing is computed

Optional but recommended:

- consider whether showing file counts beside populated systems adds value now that the data is available
- if not used now, keep the UI unchanged and reserve `filecount` for a later iteration

### 5. Keep `contCol` Logic Focused on Game Listing

`webapp/src/views/columns/ContentColumn.tsx` should continue to fetch actual game names via `GET /retropie/{system}` after selection.

No change in intent:

- selected system drives content list
- empty systems should show the normal empty list state
- upload into an empty system should still be supported
- download/delete/upload behavior stays as-is

This keeps `GET /retropie/{system}` as the source of truth for actual content rows while `/cartridgeinfo` becomes the source of truth for system bucketing.

### 6. Keep `cartCol` Compatible With The New Type

`webapp/src/views/columns/CartridgeColumn.tsx` does not appear to need behavior changes, but its local `CartridgeInfo` type must be updated to match the generated API shape.

This is primarily a typing/compatibility cleanup step.

### 7. Validate With Generated API Types

The webapp already generates API types from the OpenAPI spec in `webapp/src/api/client.ts`.

After the code changes:

- regenerated schema types must flow through cleanly
- no local type shims should reintroduce the old payload shape
- the app should compile without pretending the backend still returns `string[]` for `systems`

## Suggested Code Areas

Primary files likely involved:

- `webapp/src/App.tsx`
- `webapp/src/views/columns/OSColumn.tsx`
- `webapp/src/views/columns/ContentColumn.tsx`
- `webapp/src/views/columns/CartridgeColumn.tsx`
- `webapp/src/api/client.ts`

## Validation Checklist

1. `npm run typecheck` passes.
2. `npm run build` passes.
3. `osCol` shows populated systems without issuing one request per system for bucketing.
4. `Add Game System` reveals backend-provided empty systems.
5. Selecting an empty system still opens `contCol` and shows the expected empty state.
6. Selecting a populated system still loads and displays game rows correctly.
7. Uploading into an empty system works and the UI refresh path remains coherent.
8. Narrow-mode navigation still allows full completion of the main flows.
9. No previous clipping, hidden-tail-action, or list-flicker regressions return.

## Out Of Scope

- changing the 3-column layout
- redesigning the visual language
- changing `/retropie` endpoint semantics
- introducing a new generic filesystem browser
- adding advanced filtering or bulk actions
- changing backend API contracts again from the webapp side

## Recommended Notes For The Webapp Team

Two practical points are worth calling out for whoever implements this:

- The old system-bucketing logic is now technical debt and should be removed, not adapted.
- `filecount` should be treated as trustworthy backend metadata, but it does not replace `GET /retropie/{system}` for loading actual game names.