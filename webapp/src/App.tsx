import styles from './App.module.css';
import { MainView } from './views/MainView';
import { RetroPieView } from './views/RetroPieView';
import { useState } from 'react';

type AppMode = 'main' | 'retropie';

export function App() {
  const [mode, setMode] = useState<AppMode>('main');

  if (mode === 'retropie') {
    return (
      <main className={styles.container}>
        <RetroPieView
          onBack={() => {
            setMode('main');
          }}
        />
      </main>
    );
  }

  return (
    <main className={styles.container}>
      <MainView
        pollIntervalMs={1000}
        onManageRetroPieClick={() => {
          setMode('retropie');
        }}
      />
    </main>
  );
}
