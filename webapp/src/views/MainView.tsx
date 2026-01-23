import { useEffect, useMemo, useRef, useState, type ChangeEvent } from 'react';
import { APIError, ejectCartridge, getCartridgeInfo } from '../api';

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
  onManageRetroPieClick?: () => void;
};

type EjectFlowState =
  | { kind: 'idle' }
  | { kind: 'requesting'; sawNotPresent: boolean }
  | { kind: 'active'; sawNotPresent: boolean }
  | { kind: 'error'; message: string };

type FlashFlowState =
  | { kind: 'idle' }
  | { kind: 'picking' }
  | { kind: 'uploading'; filename: string; percent: number }
  | { kind: 'done'; filename: string }
  | { kind: 'error'; message: string };

export function MainView({ pollIntervalMs = 1000, onManageRetroPieClick }: MainViewProps) {
  const [state, setState] = useState<MainViewState>({ kind: 'loading' });
  const [lastUpdatedAt, setLastUpdatedAt] = useState<Date | null>(null);
  const [ejectFlow, setEjectFlow] = useState<EjectFlowState>({ kind: 'idle' });
  const [flashFlow, setFlashFlow] = useState<FlashFlowState>({ kind: 'idle' });

  const flashFileInputRef = useRef<HTMLInputElement | null>(null);
  const flashXhrRef = useRef<XMLHttpRequest | null>(null);

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

  useEffect(() => {
    if (!currentInfo) {
      return;
    }

    if (ejectFlow.kind !== 'requesting' && ejectFlow.kind !== 'active') {
      return;
    }

    const presentNow = currentInfo.present;
    if (!presentNow && !ejectFlow.sawNotPresent) {
      setEjectFlow({ kind: 'active', sawNotPresent: true });
      return;
    }

    // Exit eject flow once we've observed removal and the next cartridge is present.
    if (presentNow && ejectFlow.sawNotPresent && !currentInfo.busy) {
      setEjectFlow({ kind: 'idle' });
    }
  }, [currentInfo, ejectFlow]);

  const actionState = useMemo(() => {
    const busy = currentInfo?.busy ?? false;
    const present = currentInfo?.present ?? false;
    const isRetroPie = currentInfo?.isRetroPie ?? false;

    const inEjectFlow = ejectFlow.kind !== 'idle' && ejectFlow.kind !== 'error';
    const inFlashFlow = flashFlow.kind === 'picking' || flashFlow.kind === 'uploading';

    return {
      ejectDisabled: !present || busy,
      flashDisabled: !present || busy || inEjectFlow || inFlashFlow,
      manageDisabled: !present || busy || !isRetroPie || inEjectFlow || inFlashFlow
    };
  }, [currentInfo, ejectFlow, flashFlow]);

  const handleEjectClick = async () => {
    setEjectFlow({ kind: 'requesting', sawNotPresent: false });
    try {
      await ejectCartridge();
      setEjectFlow({ kind: 'active', sawNotPresent: false });
    } catch (error) {
      setEjectFlow({ kind: 'error', message: formatErrorMessage(error) });
    }
  };

  const openFlashPicker = () => {
    if (flashFlow.kind === 'uploading') {
      return;
    }
    setFlashFlow({ kind: 'picking' });
    flashFileInputRef.current?.click();
  };

  const cancelFlash = () => {
    const xhr = flashXhrRef.current;
    if (xhr) {
      xhr.abort();
    }
    flashXhrRef.current = null;
    setFlashFlow({ kind: 'idle' });
  };

  const startFlashUpload = (file: File) => {
    const normalizedName = file.name.toLowerCase();
    if (!normalizedName.endsWith('.img.gz')) {
      setFlashFlow({ kind: 'error', message: 'Only .img.gz files are allowed' });
      return;
    }
    if (file.size <= 0) {
      setFlashFlow({ kind: 'error', message: 'Selected file is empty' });
      return;
    }

    const xhr = new XMLHttpRequest();
    flashXhrRef.current = xhr;

    xhr.open('POST', '/api/v1/flash');
    xhr.setRequestHeader('Content-Type', 'application/gzip');

    xhr.upload.onprogress = (event) => {
      if (!event.lengthComputable) {
        return;
      }
      const percent = Math.max(0, Math.min(100, (event.loaded / event.total) * 100));
      setFlashFlow({ kind: 'uploading', filename: file.name, percent });
    };

    xhr.onload = () => {
      const isOk = xhr.status >= 200 && xhr.status < 300;
      if (isOk) {
        setFlashFlow({ kind: 'done', filename: file.name });
      } else {
        const message = xhr.responseText ? xhr.responseText : `HTTP ${xhr.status}`;
        setFlashFlow({ kind: 'error', message });
      }
      flashXhrRef.current = null;
    };

    xhr.onerror = () => {
      setFlashFlow({ kind: 'error', message: 'Network error during upload' });
      flashXhrRef.current = null;
    };

    xhr.onabort = () => {
      // cancelFlash() handles state; keep this for completeness.
      flashXhrRef.current = null;
    };

    setFlashFlow({ kind: 'uploading', filename: file.name, percent: 0 });
    xhr.send(file);
  };

  const handleFlashFilePicked = (event: ChangeEvent<HTMLInputElement>) => {
    const files = event.target.files;
    const file = files && files.length > 0 ? files[0] : null;

    // Reset the input so picking the same file again re-triggers onChange.
    event.target.value = '';

    if (!file) {
      setFlashFlow({ kind: 'idle' });
      return;
    }

    startFlashUpload(file);
  };

  return (
    <div>
      <h1>Keymaker</h1>

      {state.kind === 'loading' && <p>Loading cartridge info…</p>}

      {state.kind === 'error' && (
        <p>
          <strong>Error:</strong> {state.message}
        </p>
      )}

      {ejectFlow.kind !== 'idle' && (
        <section>
          <h2>Eject</h2>
          {ejectFlow.kind === 'requesting' && <p>Requesting ejection…</p>}
          {ejectFlow.kind === 'active' && (
            <p>
              Ejection initiated. Please remove the cartridge, then insert a different one.
            </p>
          )}
          {ejectFlow.kind === 'error' && (
            <p>
              <strong>Eject failed:</strong> {ejectFlow.message}
            </p>
          )}
        </section>
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
        <input
          ref={flashFileInputRef}
          type="file"
          accept=".img.gz,application/gzip"
          style={{ display: 'none' }}
          onChange={handleFlashFilePicked}
        />

        <button
          type="button"
          onClick={handleEjectClick}
          disabled={
            actionState.ejectDisabled ||
            ejectFlow.kind === 'requesting' ||
            ejectFlow.kind === 'active' ||
            flashFlow.kind === 'picking' ||
            flashFlow.kind === 'uploading'
          }
        >
          Eject cartridge
        </button>
        <button type="button" onClick={openFlashPicker} disabled={actionState.flashDisabled}>
          Flash cartridge (.img.gz)
        </button>
        <button type="button" onClick={onManageRetroPieClick} disabled={actionState.manageDisabled}>
          Manage RetroPie games
        </button>
      </section>

      {flashFlow.kind !== 'idle' && (
        <section>
          <h2>Flash</h2>
          {flashFlow.kind === 'picking' && <p>Please select a .img.gz file…</p>}
          {flashFlow.kind === 'uploading' && (
            <p>
              Uploading {flashFlow.filename}: {Math.floor(flashFlow.percent)}%
            </p>
          )}
          {flashFlow.kind === 'done' && <p>Flash finished: {flashFlow.filename}</p>}
          {flashFlow.kind === 'error' && (
            <p>
              <strong>Flash failed:</strong> {flashFlow.message}
            </p>
          )}

          {(flashFlow.kind === 'picking' || flashFlow.kind === 'uploading') && (
            <button type="button" onClick={cancelFlash}>
              Cancel
            </button>
          )}
        </section>
      )}
    </div>
  );
}
