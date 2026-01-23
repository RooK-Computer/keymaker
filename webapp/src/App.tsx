import styles from './App.module.css';
import { MainView } from './views/MainView';

export function App() {
  return (
    <main className={styles.container}>
      <MainView
        pollIntervalMs={1000}
        onEjectClick={() => {
          window.alert('Not implemented yet (step 4).');
        }}
        onFlashClick={() => {
          window.alert('Not implemented yet (step 5).');
        }}
        onManageRetroPieClick={() => {
          window.alert('Not implemented yet (step 6).');
        }}
      />
    </main>
  );
}
