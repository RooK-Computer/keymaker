import { useEffect, useRef, useState, type ChangeEvent } from 'react';
import { APIError, apiV1Url, ejectCartridge } from '../../api';
import cartridgeImage from '../../../images/cartrighe-illu.png';
import styles from './FlasherColumns.module.css';

type CartridgeInfo = {
  present: boolean;
  mounted: boolean;
  isRetroPie: boolean;
  systems: string[] | null;
  busy: boolean;
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

export type CartridgeColumnProps = {
  info: CartridgeInfo | null;
};

export function CartridgeColumn({ info }: CartridgeColumnProps) {
  const [ejectFlow, setEjectFlow] = useState<EjectFlowState>({ kind: 'idle' });
  const [flashFlow, setFlashFlow] = useState<FlashFlowState>({ kind: 'idle' });
  const flashFileInputRef = useRef<HTMLInputElement | null>(null);
  const flashXhrRef = useRef<XMLHttpRequest | null>(null);

  const busy = info?.busy ?? false;
  const present = info?.present ?? false;
  const inEjectFlow = ejectFlow.kind === 'requesting' || ejectFlow.kind === 'active';
  const inFlashFlow = flashFlow.kind === 'picking' || flashFlow.kind === 'uploading';

  const ejectDisabled = !present || busy || inFlashFlow || inEjectFlow;
  const flashDisabled = !present || busy || inEjectFlow || inFlashFlow;

  const handleEjectClick = async () => {
    setEjectFlow({ kind: 'requesting', sawNotPresent: false });
    try {
      await ejectCartridge();
      setEjectFlow({ kind: 'active', sawNotPresent: false });
    } catch (error) {
      setEjectFlow({ kind: 'error', message: formatErrorMessage(error) });
    }
  };

  useEffect(() => {
    if (!info || !inEjectFlow) {
      return;
    }
    if (!info.present && !ejectFlow.sawNotPresent) {
      setEjectFlow({ kind: 'active', sawNotPresent: true });
    } else if (info.present && ejectFlow.sawNotPresent && !info.busy) {
      setEjectFlow({ kind: 'idle' });
    }
  }, [ejectFlow, inEjectFlow, info]);

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

    xhr.open('POST', apiV1Url('/flash'));
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
      flashXhrRef.current = null;
    };

    setFlashFlow({ kind: 'uploading', filename: file.name, percent: 0 });
    xhr.send(file);
  };

  const handleFlashFilePicked = (event: ChangeEvent<HTMLInputElement>) => {
    const files = event.target.files;
    const file = files && files.length > 0 ? files[0] : null;
    event.target.value = '';

    if (!file) {
      setFlashFlow({ kind: 'idle' });
      return;
    }

    startFlashUpload(file);
  };

  return (
    <section className={styles.section}>
      <h2 className={styles.title}>Cartridge</h2>

      {!info ? (
        <p className={styles.emptyText}>No data yet.</p>
      ) : (
        <>
          {info.present && (
            <div className={styles.cartridgeImageWrap}>
              <img className={styles.cartridgeImage} src={cartridgeImage} alt="Cartridge" />
            </div>
          )}
          <p className={styles.statusLine}>{info.present ? (info.isRetroPie ? 'RetroPie' : 'Unknown OS') : 'No cartridge'}</p>
          <p className={styles.statusMeta}>Mounted: {String(info.mounted)} | Busy: {String(info.busy)}</p>
        </>
      )}

      <input
        ref={flashFileInputRef}
        type="file"
        accept=".img.gz,application/gzip"
        style={{ display: 'none' }}
        onChange={handleFlashFilePicked}
      />

      <div className={styles.actionArea}>
        <button className={styles.pixelButton} type="button" onClick={openFlashPicker} disabled={flashDisabled}>
          Flash Cartridge
        </button>
        <button className={styles.pixelButton} type="button" onClick={handleEjectClick} disabled={ejectDisabled}>
          Rebottle Cartridge
        </button>
      </div>

      {ejectFlow.kind === 'requesting' && <p className={styles.subtleMessage}>Requesting ejection…</p>}
      {ejectFlow.kind === 'active' && <p className={styles.subtleMessage}>Please remove cartridge and insert the next one.</p>}
      {ejectFlow.kind === 'error' && (
        <p className={styles.errorMessage}>
          Eject failed: {ejectFlow.message}
        </p>
      )}

      {flashFlow.kind === 'picking' && <p className={styles.flashHint}>Please select a .img.gz file…</p>}
      {flashFlow.kind === 'uploading' && (
        <p className={styles.flashHint}>
          Uploading {flashFlow.filename}: {Math.floor(flashFlow.percent)}%
        </p>
      )}
      {flashFlow.kind === 'done' && <p className={styles.flashHint}>Flash finished: {flashFlow.filename}</p>}
      {flashFlow.kind === 'error' && (
        <p className={styles.errorMessage}>
          Flash failed: {flashFlow.message}
        </p>
      )}
      {(flashFlow.kind === 'picking' || flashFlow.kind === 'uploading') && (
        <button className={styles.miniButton} type="button" onClick={cancelFlash}>
          Cancel
        </button>
      )}
    </section>
  );
}
