import { useEffect, useMemo, useState } from 'react';
import { APIError, getCartridgeInfo } from '../api';

type CartridgeInfo = Awaited<ReturnType<typeof getCartridgeInfo>>;

type MainViewState =
  | { kind: 'loading'; lastInfo?: CartridgeInfo }
  | { kind: 'ready'; info: CartridgeInfo }
  | { kind: 'error'; message: string; lastInfo?: CartridgeInfo };

function formatErrorMessage(error: unknown): string {
  if (error instanceof APIError) {
    const code = error.code ? ` (${error.code})` : '';
    return `API error ${error.status}${code}: ${error.message}`;
  }
  if (error instanceof Error) {
    return error.message;
  }
  try {
    return JSON.stringify(error);
  } catch {
    return String(error);
  }
}

export type MainViewProps = {
  pollIntervalMs?: number;
  onEjectClick?: () => void;
  onFlashClick?: () => void;
  onManageRetroPieClick?: () => void;
};

export function MainView({
  pollIntervalMs = 1000,
  onEjectClick,
  onFlashClick,
  onManageRetroPieClick
}: MainViewProps) {
  const [state, setState] = useState<MainViewState>({ kind: 'loading' });
  const [lastUpdatedAt, setLastUpdatedAt] = useState<Date | null>(null);

  useEffect(() => {
    let cancelled = false;
    const abortController = new AbortController();
    let timeoutId: number | null = null;

    const tick = async () => {
      try {
        const info = await getCartridgeInfo(abortController.signal);
        if (cancelled) {
          return;
        }
        setState({ kind: 'ready', info });
        setLastUpdatedAt(new Date());
      } catch (error) {
        if (cancelled) {
          return;
        }
        // Ignore abort on unmount.
        if (error instanceof DOMException && error.name === 'AbortError') {
          return;
        }

        const message = formatErrorMessage(error);
        setState((prev) => {
          const lastInfo = prev.kind === 'ready' ? prev.info : prev.lastInfo;
          return { kind: 'error', message, lastInfo };
        });
      } finally {
        if (cancelled) {
          return;
        }
        timeoutId = window.setTimeout(tick, pollIntervalMs);
      }
    };

    tick();

    return () => {
      cancelled = true;
      abortController.abort();
      if (timeoutId !== null) {
        window.clearTimeout(timeoutId);
      }
    };
  }, [pollIntervalMs]);

  const currentInfo = useMemo(() => {
    if (state.kind === 'ready') {
      return state.info;
    }
    return state.lastInfo;
  }, [state]);

  const actionState = useMemo(() => {
    const busy = currentInfo?.busy ?? false;
    const present = currentInfo?.present ?? false;
    const isRetroPie = currentInfo?.isRetroPie ?? false;

    return {
      ejectDisabled: !present || busy,
      flashDisabled: !present || busy,
      manageDisabled: !present || busy || !isRetroPie
    };
  }, [currentInfo]);

  return (
    <div>
      <h1>Keymaker</h1>

      {state.kind === 'loading' && <p>Loading cartridge info…</p>}

      {state.kind === 'error' && (
        <p>
          <strong>Error:</strong> {state.message}
        </p>
      )}

      <section>
        <h2>Cartridge</h2>
        {!currentInfo ? (
          <p>No data yet.</p>
        ) : (
          <dl>
            <dt>Present</dt>
            <dd>{String(currentInfo.present)}</dd>

            <dt>Mounted</dt>
            <dd>{String(currentInfo.mounted)}</dd>

            <dt>Busy</dt>
            <dd>{String(currentInfo.busy)}</dd>

            <dt>RetroPie</dt>
            <dd>{String(currentInfo.isRetroPie)}</dd>

            <dt>Systems</dt>
            <dd>{currentInfo.systems.length ? currentInfo.systems.join(', ') : '(none)'}</dd>
          </dl>
        )}

        <p>
          Polling every {Math.round(pollIntervalMs / 100) / 10}s
          {lastUpdatedAt ? ` — last updated ${lastUpdatedAt.toLocaleTimeString()}` : ''}
        </p>
      </section>

      <section>
        <h2>Actions</h2>
        <button type="button" onClick={onEjectClick} disabled={actionState.ejectDisabled}>
          Eject cartridge
        </button>
        <button type="button" onClick={onFlashClick} disabled={actionState.flashDisabled}>
          Flash cartridge (.img.gz)
        </button>
        <button type="button" onClick={onManageRetroPieClick} disabled={actionState.manageDisabled}>
          Manage RetroPie games
        </button>
      </section>
    </div>
  );
}
