import { useState } from 'react';
import styles from './FlasherColumns.module.css';

type CartridgeInfo = {
  isRetroPie: boolean;
  systems: string[] | null;
  present: boolean;
};

export type OSColumnProps = {
  info: CartridgeInfo | null;
  systems: string[];
  emptySystems: string[];
  systemsLoading: boolean;
  selectedSystem: string | null;
  onSelectSystem: (system: string) => void;
};

export function OSColumn({ info, systems, emptySystems, systemsLoading, selectedSystem, onSelectSystem }: OSColumnProps) {
  const [showEmptySystems, setShowEmptySystems] = useState(false);
  const hasSystems = systems.length > 0;

  return (
    <section className={styles.section}>
      <h2 className={styles.title}>System</h2>

      {!info || !info.present ? (
        <p className={styles.emptyText}>No cartridge detected.</p>
      ) : (
        <>
          <p className={styles.subtleMessage}>Detected: {info.isRetroPie ? 'RetroPie' : 'Unknown / unsupported'}</p>
          {systemsLoading && <p className={styles.subtleMessage}>Checking emulator game libraries…</p>}
          {!hasSystems ? (
            <p className={styles.emptyText}>(none)</p>
          ) : (
            <ul className={styles.systemsList}>
              {systems.map((system) => (
                <li className={styles.systemRow} key={system}>
                  <button
                    type="button"
                    className={`${styles.systemButton} ${selectedSystem === system ? styles.systemButtonActive : ''}`}
                    onClick={() => {
                      onSelectSystem(system);
                    }}
                    aria-pressed={selectedSystem === system}
                  >
                    {system}
                  </button>
                </li>
              ))}
            </ul>
          )}

          <div className={styles.listTailAction}>
            <button
              type="button"
              className={styles.pixelButton}
              onClick={() => {
                setShowEmptySystems((prev) => !prev);
              }}
              disabled={emptySystems.length === 0}
            >
              Add Game System
            </button>

            {showEmptySystems && emptySystems.length > 0 && (
              <ul className={styles.systemsList}>
                {emptySystems.map((system) => (
                  <li className={styles.systemRow} key={`empty-${system}`}>
                    <button
                      type="button"
                      className={`${styles.systemButton} ${selectedSystem === system ? styles.systemButtonActive : ''}`}
                      onClick={() => {
                        onSelectSystem(system);
                      }}
                      aria-pressed={selectedSystem === system}
                    >
                      {system}
                    </button>
                  </li>
                ))}
              </ul>
            )}
            {showEmptySystems && emptySystems.length === 0 && <p className={styles.subtleMessage}>No empty systems available.</p>}
          </div>
        </>
      )}
    </section>
  );
}
