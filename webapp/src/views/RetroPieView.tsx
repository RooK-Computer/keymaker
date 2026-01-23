import { useEffect, useMemo, useRef, useState, type ChangeEvent } from 'react';
import {
  APIError,
  deleteRetroPieGame,
  listRetroPieGames,
  listRetroPieSystems,
  uploadRetroPieGame
} from '../api';

type RetroPieViewState =
  | { kind: 'loading' }
  | { kind: 'ready'; systems: string[] }
  | { kind: 'error'; message: string };

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

export type RetroPieViewProps = {
  onBack: () => void;
};

export function RetroPieView({ onBack }: RetroPieViewProps) {
  const [state, setState] = useState<RetroPieViewState>({ kind: 'loading' });
  const [selectedSystem, setSelectedSystem] = useState<string | null>(null);
  const [gamesState, setGamesState] = useState<GamesState>({ kind: 'idle' });
  const [uploadState, setUploadState] = useState<UploadState>({ kind: 'idle' });
  const [deleteState, setDeleteState] = useState<{ kind: 'idle' } | { kind: 'deleting'; game: string } | { kind: 'error'; message: string }>({
    kind: 'idle'
  });

  const uploadFileInputRef = useRef<HTMLInputElement | null>(null);
  const uploadAbortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    const abortController = new AbortController();

    (async () => {
      try {
        const systems = await listRetroPieSystems(abortController.signal);
        setState({ kind: 'ready', systems });
      } catch (error) {
        if (error instanceof DOMException && error.name === 'AbortError') {
          return;
        }
        setState({ kind: 'error', message: formatErrorMessage(error) });
      }
    })();

    return () => {
      abortController.abort();
    };
  }, []);

  useEffect(() => {
    if (!selectedSystem) {
      setGamesState({ kind: 'idle' });
      setUploadState({ kind: 'idle' });
      return;
    }

    const abortController = new AbortController();
    setGamesState({ kind: 'loading', system: selectedSystem });

    (async () => {
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

  const systems = useMemo(() => (state.kind === 'ready' ? state.systems : []), [state]);

  const games = useMemo(() => {
    if (gamesState.kind === 'ready') {
      return gamesState.games;
    }
    return [];
  }, [gamesState]);

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
    if (!selectedSystem) {
      return;
    }
    if (uploadState.kind === 'uploading') {
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
      // The API uses {game} as the destination filename; we use the picked filename.
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

  return (
    <div>
      <h1>RetroPie</h1>

      <button type="button" onClick={onBack}>
        Back
      </button>

      {state.kind === 'loading' && <p>Loading systems…</p>}

      {state.kind === 'error' && (
        <p>
          <strong>Error:</strong> {state.message}
        </p>
      )}

      {state.kind === 'ready' && (
        <section>
          <h2>Systems</h2>
          {systems.length === 0 ? (
            <p>(none)</p>
          ) : (
            <div>
              {systems.map((system) => (
                <div key={system}>
                  <button type="button" onClick={() => setSelectedSystem(system)}>
                    {system}
                  </button>
                </div>
              ))}
            </div>
          )}

          {selectedSystem && (
            <section>
              <h2>Selected system</h2>
              <p>{selectedSystem}</p>

              <h3>Upload</h3>
              <input
                ref={uploadFileInputRef}
                type="file"
                style={{ display: 'none' }}
                onChange={handleUploadFilePicked}
              />
              <button type="button" onClick={openUploadPicker} disabled={uploadState.kind === 'uploading'}>
                Upload game
              </button>
              {uploadState.kind === 'picking' && <p>Please select a file…</p>}
              {uploadState.kind === 'uploading' && <p>Uploading {uploadState.filename}…</p>}
              {uploadState.kind === 'done' && <p>Upload completed: {uploadState.filename}</p>}
              {uploadState.kind === 'error' && (
                <p>
                  <strong>Upload failed:</strong> {uploadState.message}
                </p>
              )}
              {uploadState.kind === 'uploading' && (
                <button type="button" onClick={cancelUpload}>
                  Cancel upload
                </button>
              )}

              <h3>Games</h3>
              {gamesState.kind === 'loading' && <p>Loading games…</p>}
              {gamesState.kind === 'error' && (
                <p>
                  <strong>Error:</strong> {gamesState.message}
                </p>
              )}
              {deleteState.kind === 'error' && (
                <p>
                  <strong>Delete failed:</strong> {deleteState.message}
                </p>
              )}
              {gamesState.kind === 'ready' && games.length === 0 && <p>(none)</p>}
              {gamesState.kind === 'ready' && games.length > 0 && (
                <ul>
                  {games.map((game) => {
                    const url = `/api/v1/retropie/${encodeURIComponent(selectedSystem)}/${encodeURIComponent(game)}`;
                    return (
                      <li key={game}>
                        {game} —{' '}
                        <a href={url} target="_blank" rel="noreferrer">
                          download
                        </a>
                        {' '}|{' '}
                        <button
                          type="button"
                          onClick={() => {
                            void deleteGame(selectedSystem, game);
                          }}
                          disabled={deleteState.kind === 'deleting'}
                        >
                          {deleteState.kind === 'deleting' && deleteState.game === game ? 'deleting…' : 'delete'}
                        </button>
                      </li>
                    );
                  })}
                </ul>
              )}
            </section>
          )}
        </section>
      )}
    </div>
  );
}
