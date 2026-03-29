import { useEffect, useMemo, useState } from 'react';
import { APIError, getCartridgeInfo, listRetroPieGames } from './api';
import styles from './App.module.css';
import { CartridgeColumn } from './views/columns/CartridgeColumn';
import { ContentColumn } from './views/columns/ContentColumn';
import { OSColumn } from './views/columns/OSColumn';

export function App() {
  const [infoState, setInfoState] = useState<
    | { kind: 'loading'; info: null; error: null }
    | { kind: 'ready'; info: Awaited<ReturnType<typeof getCartridgeInfo>>; error: null }
    | { kind: 'error'; info: Awaited<ReturnType<typeof getCartridgeInfo>> | null; error: string }
  >({ kind: 'loading', info: null, error: null });
  const [activeColumn, setActiveColumn] = useState<'cart' | 'os' | 'content'>('cart');
  const [selectedSystem, setSelectedSystem] = useState<string | null>(null);
  const [systemAvailability, setSystemAvailability] = useState<{
    withGames: string[];
    withoutGames: string[];
    loading: boolean;
  }>({ withGames: [], withoutGames: [], loading: false });
  const [isNarrow, setIsNarrow] = useState<boolean>(window.innerWidth < 1100);

  useEffect(() => {
    const onResize = () => {
      setIsNarrow(window.innerWidth < 1100);
    };
    window.addEventListener('resize', onResize);
    return () => {
      window.removeEventListener('resize', onResize);
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    const abortController = new AbortController();
    let timeoutId: number | null = null;

    const formatError = (error: unknown): string => {
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
    };

    const tick = async () => {
      try {
        const info = await getCartridgeInfo(abortController.signal);
        if (cancelled) {
          return;
        }
        setInfoState({ kind: 'ready', info, error: null });
      } catch (error) {
        if (cancelled) {
          return;
        }
        if (error instanceof DOMException && error.name === 'AbortError') {
          return;
        }
        setInfoState((prev) => ({
          kind: 'error',
          info: prev.info,
          error: formatError(error)
        }));
      } finally {
        if (cancelled) {
          return;
        }
        timeoutId = window.setTimeout(tick, 1000);
      }
    };

    void tick();

    return () => {
      cancelled = true;
      abortController.abort();
      if (timeoutId !== null) {
        window.clearTimeout(timeoutId);
      }
    };
  }, []);

  const info = infoState.info;
  const systemsKey = useMemo(() => (info?.systems ?? []).join('|'), [info?.systems]);
  const osResetKey = useMemo(() => {
    if (!info) {
      return 'none';
    }
    return `${info.present}:${info.isRetroPie}:${(info.systems ?? []).join('|')}:${info.busy}`;
  }, [info]);

  const contentResetKey = osResetKey;

  const shouldShow = (column: 'cart' | 'os' | 'content') => {
    if (!isNarrow) {
      return true;
    }
    return activeColumn === column;
  };

  useEffect(() => {
    if (!isNarrow) {
      return;
    }
    if (!info?.present) {
      setActiveColumn('cart');
      return;
    }
    if (info.isRetroPie) {
      setActiveColumn('content');
      return;
    }
    setActiveColumn('os');
  }, [info?.present, info?.isRetroPie, isNarrow]);

  const columnClass = (column: 'cart' | 'os' | 'content') =>
    shouldShow(column) ? styles.columnVisible : styles.columnHidden;

  const unavailableContent = !!info?.present && !info.isRetroPie;

  useEffect(() => {
    const systems = info?.systems ?? [];
    if (!info?.present || !info.isRetroPie || systems.length === 0) {
      setSystemAvailability({ withGames: systems, withoutGames: [], loading: false });
      return;
    }

    let cancelled = false;
    setSystemAvailability((prev) => ({ ...prev, loading: true }));

    void (async () => {
      const prevMap = new Map<string, boolean>();
      for (const s of systemAvailability.withGames) {
        prevMap.set(s, true);
      }
      for (const s of systemAvailability.withoutGames) {
        prevMap.set(s, false);
      }

      const results = await Promise.all(
        systems.map(async (system) => {
          try {
            const games = await listRetroPieGames(system);
            return { system, hasGames: games.length > 0 };
          } catch {
            // Keep prior classification to avoid UI flicker on transient failures.
            return { system, hasGames: prevMap.get(system) ?? false };
          }
        })
      );

      if (cancelled) {
        return;
      }

      const withGames = results.filter((r) => r.hasGames).map((r) => r.system);
      const withoutGames = results.filter((r) => !r.hasGames).map((r) => r.system);
      setSystemAvailability({ withGames, withoutGames, loading: false });
    })();

    return () => {
      cancelled = true;
    };
  }, [info?.present, info?.isRetroPie, systemsKey]);

  useEffect(() => {
    if (!info?.present || !info.isRetroPie) {
      setSelectedSystem(null);
      return;
    }
    const fallbackSystems = info.systems ?? [];
    const systems = systemAvailability.withGames.length > 0 ? systemAvailability.withGames : fallbackSystems;
    if (!selectedSystem || !systems.includes(selectedSystem)) {
      setSelectedSystem(systems.length > 0 ? systems[0] : null);
    }
  }, [info?.present, info?.isRetroPie, info?.systems, selectedSystem, systemAvailability.withGames]);

  return (
    <main className={styles.container}>
      <header className={styles.header}>
        <h1>Flasher</h1>
        {infoState.kind === 'loading' && <p>Loading cartridge info…</p>}
        {infoState.kind === 'error' && <p>Error: {infoState.error}</p>}
      </header>
      {isNarrow && (
        <nav className={styles.mobileNav}>
          <button type="button" onClick={() => setActiveColumn('cart')}>
            Cartridge
          </button>
          <button type="button" onClick={() => setActiveColumn('os')} disabled={!info?.present}>
            System
          </button>
          <button type="button" onClick={() => setActiveColumn('content')} disabled={!info?.present || unavailableContent}>
            Games
          </button>
        </nav>
      )}
      <section className={styles.columns}>
        <article className={`${styles.column} ${columnClass('cart')}`}>
          <CartridgeColumn info={info} />
        </article>
        <article className={`${styles.column} ${columnClass('os')}`}>
          <OSColumn
            info={info}
            systems={systemAvailability.withGames}
            emptySystems={systemAvailability.withoutGames}
            systemsLoading={systemAvailability.loading}
            selectedSystem={selectedSystem}
            onSelectSystem={(system) => {
              setSelectedSystem(system);
              if (isNarrow) {
                setActiveColumn('content');
              }
            }}
          />
        </article>
        <article className={`${styles.column} ${columnClass('content')}`} key={`content:${contentResetKey}`}>
          <ContentColumn info={info} resetKey={contentResetKey} selectedSystem={selectedSystem} />
        </article>
      </section>
    </main>
  );
}
