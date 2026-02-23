import { check, Update } from "@tauri-apps/plugin-updater";
import { relaunch } from "@tauri-apps/plugin-process";
import { getVersion } from "@tauri-apps/api/app";
import { useState, useEffect, useCallback } from "react";

export type UpdaterStatus =
  | { state: "idle" }
  | { state: "checking" }
  | { state: "up-to-date" }
  | { state: "available"; update: Update }
  | { state: "downloading"; progress: number | null }
  | { state: "ready" }
  | { state: "error"; message: string };

export function useUpdater() {
  const [status, setStatus] = useState<UpdaterStatus>({ state: "idle" });
  const [currentVersion, setCurrentVersion] = useState<string>("");

  useEffect(() => {
    getVersion().then(setCurrentVersion).catch(() => {});
    checkForUpdate();
  }, []);

  const checkForUpdate = useCallback(async () => {
    setStatus({ state: "checking" });
    try {
      const update = await check();
      if (update) {
        setStatus({ state: "available", update });
      } else {
        setStatus({ state: "up-to-date" });
      }
    } catch (err) {
      setStatus({ state: "error", message: String(err) });
    }
  }, []);

  const installUpdate = useCallback(async () => {
    if (status.state !== "available") return;
    const { update } = status;

    setStatus({ state: "downloading", progress: null });

    // Track total size reported at download start to compute percentage.
    let contentLength: number | undefined;
    let downloaded = 0;

    try {
      await update.downloadAndInstall((event) => {
        switch (event.event) {
          case "Started":
            contentLength = event.data.contentLength;
            downloaded = 0;
            setStatus({ state: "downloading", progress: contentLength ? 0 : null });
            break;
          case "Progress":
            downloaded += event.data.chunkLength;
            setStatus({
              state: "downloading",
              progress: contentLength
                ? Math.min(100, Math.round((downloaded / contentLength) * 100))
                : null,
            });
            break;
          case "Finished":
            setStatus({ state: "ready" });
            break;
        }
      });

      // downloadAndInstall resolves after install is staged; relaunch now.
      await relaunch();
    } catch (err) {
      setStatus({ state: "error", message: String(err) });
    }
  }, [status]);

  return { status, currentVersion, checkForUpdate, installUpdate };
}
