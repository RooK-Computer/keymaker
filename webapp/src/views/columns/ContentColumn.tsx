import { useEffect, useRef, useState, type ChangeEvent } from 'react';
import { APIError, apiV1Url, deleteRetroPieGame, getCartridgeInfo, listRetroPieGames, uploadRetroPieGame } from '../../api';
import styles from './FlasherColumns.module.css';

const nameCollator = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' });

function sortNames(values: string[]): string[] {
  return [...values].sort(nameCollator.compare);
}

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

type GamesState =
  | { kind: 'idle' }
  | { kind: 'loading'; system: string }
  | { kind: 'ready'; system: string; games: string[] }
  | { kind: 'error'; system: string; message: string };

type UploadState =
  | { kind: 'idle' }
  | { kind: 'picking' }
  | { kind: 'uploading'; filename: string }
  | { kind: 'done'; filename: string }
  | { kind: 'error'; message: string };

type CartridgeInfo = Awaited<ReturnType<typeof getCartridgeInfo>>;

export type ContentColumnProps = {
  info: CartridgeInfo | null;
  resetKey: string;
  selectedSystem: string | null;
};

export function ContentColumn({ info, resetKey, selectedSystem }: ContentColumnProps) {
  const [gamesState, setGamesState] = useState<GamesState>({ kind: 'idle' });
  const [uploadState, setUploadState] = useState<UploadState>({ kind: 'idle' });
  const [deleteState, setDeleteState] = useState<{ kind: 'idle' } | { kind: 'deleting'; game: string } | { kind: 'error'; message: string }>({
    kind: 'idle'
  });

  const uploadFileInputRef = useRef<HTMLInputElement | null>(null);
  const uploadAbortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    setGamesState({ kind: 'idle' });
    setUploadState({ kind: 'idle' });
    setDeleteState({ kind: 'idle' });
    uploadAbortRef.current?.abort();
    uploadAbortRef.current = null;
  }, [resetKey]);

  useEffect(() => {
    if (!selectedSystem) {
      setGamesState({ kind: 'idle' });
      return;
    }
    const abortController = new AbortController();
    setGamesState({ kind: 'loading', system: selectedSystem });
    void (async () => {
      try {
        const games = await listRetroPieGames(selectedSystem, abortController.signal);
        setGamesState({ kind: 'ready', system: selectedSystem, games });
      } catch (error) {
        if (error instanceof DOMException && error.name === 'AbortError') {
          return;
        }
        setGamesState({ kind: 'error', system: selectedSystem, message: formatErrorMessage(error) });
      }
    })();

    return () => {
      abortController.abort();
    };
  }, [selectedSystem]);

  const reloadGames = async (system: string, signal?: AbortSignal) => {
    setGamesState({ kind: 'loading', system });
    try {
      const list = await listRetroPieGames(system, signal);
      setGamesState({ kind: 'ready', system, games: list });
    } catch (error) {
      if (error instanceof DOMException && error.name === 'AbortError') {
        return;
      }
      setGamesState({ kind: 'error', system, message: formatErrorMessage(error) });
    }
  };

  const openUploadPicker = () => {
    if (!selectedSystem || uploadState.kind === 'uploading') {
      return;
    }
    setUploadState({ kind: 'picking' });
    uploadFileInputRef.current?.click();
  };

  const cancelUpload = () => {
    uploadAbortRef.current?.abort();
    uploadAbortRef.current = null;
    setUploadState({ kind: 'idle' });
  };

  const handleUploadFilePicked = async (event: ChangeEvent<HTMLInputElement>) => {
    const files = event.target.files;
    const file = files && files.length > 0 ? files[0] : null;
    event.target.value = '';

    if (!file || !selectedSystem) {
      setUploadState({ kind: 'idle' });
      return;
    }
    if (file.size <= 0) {
      setUploadState({ kind: 'error', message: 'Selected file is empty' });
      return;
    }

    const abortController = new AbortController();
    uploadAbortRef.current = abortController;
    setUploadState({ kind: 'uploading', filename: file.name });

    try {
      await uploadRetroPieGame(selectedSystem, file.name, file, abortController.signal);
      setUploadState({ kind: 'done', filename: file.name });
      await reloadGames(selectedSystem);
    } catch (error) {
      if (error instanceof DOMException && error.name === 'AbortError') {
        setUploadState({ kind: 'idle' });
        return;
      }
      setUploadState({ kind: 'error', message: formatErrorMessage(error) });
    } finally {
      uploadAbortRef.current = null;
    }
  };

  const deleteGame = async (system: string, game: string) => {
    if (deleteState.kind === 'deleting') {
      return;
    }
    setDeleteState({ kind: 'deleting', game });
    try {
      await deleteRetroPieGame(system, game);
      setDeleteState({ kind: 'idle' });
      await reloadGames(system);
    } catch (error) {
      setDeleteState({ kind: 'error', message: formatErrorMessage(error) });
    }
  };

  if (!info?.present) {
    return (
      <section className={styles.section}>
        <h2 className={styles.title}>Games</h2>
        <p className={styles.emptyText}>Insert a cartridge to view content.</p>
      </section>
    );
  }

  if (!info.isRetroPie) {
    return (
      <section className={styles.section}>
        <h2 className={styles.title}>Games</h2>
        <p className={styles.emptyText}>Content tools are not available for this operating system yet.</p>
      </section>
    );
  }

  const games = gamesState.kind === 'ready' ? sortNames(gamesState.games) : [];
  const title = selectedSystem ? `${selectedSystem.toUpperCase()} Games` : 'Games';

  return (
    <section className={styles.section}>
      <div className={styles.titleRow}>
        <h2 className={styles.title}>{title}</h2>
      </div>

      {!selectedSystem && <p className={styles.emptyText}>Select a system in the System column.</p>}

      {gamesState.kind === 'idle' && <p className={styles.subtleMessage}>Select a system to load games.</p>}
      {gamesState.kind === 'loading' && <p className={styles.subtleMessage}>Loading games…</p>}
      {gamesState.kind === 'error' && (
        <p className={styles.errorMessage}>
          Error: {gamesState.message}
        </p>
      )}
      {deleteState.kind === 'error' && (
        <p className={styles.errorMessage}>
          Delete failed: {deleteState.message}
        </p>
      )}
      {gamesState.kind === 'ready' && games.length === 0 && <p className={styles.emptyText}>(none)</p>}
      {gamesState.kind === 'ready' && games.length > 0 && selectedSystem && (
        <ul className={styles.gamesList}>
          {games.map((game) => {
            const url = apiV1Url(`/retropie/${encodeURIComponent(selectedSystem)}/${encodeURIComponent(game)}`);
            return (
              <li className={styles.gameRow} key={game}>
                <span className={styles.gameName}>{game}</span>
                <span className={styles.gameActions}>
                  <a className={styles.linkButton} href={url} target="_blank" rel="noreferrer">
                    Download
                  </a>
                  <button
                  className={styles.miniButton}
                  type="button"
                  onClick={() => {
                    void deleteGame(selectedSystem, game);
                  }}
                  disabled={deleteState.kind === 'deleting' || info.busy}
                >
                  {deleteState.kind === 'deleting' && deleteState.game === game ? 'Deleting…' : 'Delete'}
                  </button>
                </span>
              </li>
            );
          })}
        </ul>
      )}

      {selectedSystem && (
        <>
          <input ref={uploadFileInputRef} type="file" style={{ display: 'none' }} onChange={handleUploadFilePicked} />
          <div className={styles.listTailAction}>
            <button className={styles.pixelButton} type="button" onClick={openUploadPicker} disabled={uploadState.kind === 'uploading' || info.busy}>
              Add Game
            </button>
          </div>
          {uploadState.kind === 'picking' && <p className={styles.subtleMessage}>Please select a file…</p>}
          {uploadState.kind === 'uploading' && <p className={styles.subtleMessage}>Uploading {uploadState.filename}…</p>}
          {uploadState.kind === 'done' && <p className={styles.subtleMessage}>Upload completed: {uploadState.filename}</p>}
          {uploadState.kind === 'error' && (
            <p className={styles.errorMessage}>
              Upload failed: {uploadState.message}
            </p>
          )}
          {uploadState.kind === 'uploading' && (
            <button className={styles.miniButton} type="button" onClick={cancelUpload}>
              Cancel upload
            </button>
          )}
        </>
      )}
    </section>
  );
}
