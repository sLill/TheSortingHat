import { useUpdater } from "./useUpdater";
import "./App.css";

function UpdateBanner() {
  const { status, currentVersion, checkForUpdate, installUpdate } =
    useUpdater();

  return (
    <div className="update-banner">
      <p className="version-label">Current version: {currentVersion || "…"}</p>

      {status.state === "checking" && (
        <p className="status-text checking">Checking for updates…</p>
      )}

      {status.state === "up-to-date" && (
        <div className="status-block up-to-date">
          <p>You are running the latest version.</p>
          <button className="btn secondary" onClick={checkForUpdate}>
            Check again
          </button>
        </div>
      )}

      {status.state === "available" && (
        <div className="status-block available">
          <h2>Update available: {status.update.version}</h2>
          {status.update.body && (
            <pre className="release-notes">{status.update.body}</pre>
          )}
          <div className="button-row">
            <button className="btn primary" onClick={installUpdate}>
              Install and restart
            </button>
            <button className="btn secondary" onClick={checkForUpdate}>
              Dismiss
            </button>
          </div>
        </div>
      )}

      {status.state === "downloading" && (
        <div className="status-block downloading">
          <p>Downloading update…</p>
          {status.progress != null ? (
            <>
              <progress value={status.progress} max={100} />
              <span>{status.progress}%</span>
            </>
          ) : (
            <progress />
          )}
        </div>
      )}

      {status.state === "ready" && (
        <div className="status-block ready">
          <p>Update installed. Restarting…</p>
        </div>
      )}

      {status.state === "error" && (
        <div className="status-block error">
          <p>Update check failed: {status.message}</p>
          <button className="btn secondary" onClick={checkForUpdate}>
            Retry
          </button>
        </div>
      )}
    </div>
  );
}

function App() {
  return (
    <main className="container">
      <h1>The Sorting Hat</h1>
      <UpdateBanner />
    </main>
  );
}

export default App;
