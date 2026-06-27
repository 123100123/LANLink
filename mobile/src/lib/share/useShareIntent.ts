import { useEffect } from "react";

import { useSessionStore } from "@/store/sessionStore";
import {
  addShareListener,
  getInitialShareIntent,
} from "../../../modules/lanlink-share";
import { flushPendingShares, receiveSharedFiles } from "./shareInbox";

/**
 * Wires the Android system share sheet into the app. Mount once at the root.
 *
 * - Cold start (app launched from a share): reads the launch intent after the
 *   session has hydrated, so routing knows whether a device is paired.
 * - Warm start (app already open): subscribes to the native onShare event.
 * - Flushes any files that were shared before pairing the moment credentials
 *   become available.
 *
 * No-op on iOS / web / Expo Go (the native module isn't present there).
 */
export function useShareIntent(): void {
  const hydrated = useSessionStore((s) => s.hydrated);

  useEffect(() => {
    if (!hydrated) return;
    let active = true;

    // Defer the cold-start handoff one frame so it lands after the root
    // navigator's initial redirect (index.tsx) rather than racing it.
    const handle = requestAnimationFrame(() => {
      getInitialShareIntent()
        .then((files) => {
          if (active && files.length > 0) receiveSharedFiles(files);
        })
        .catch(() => {
          // A failed read just means no shared files to handle.
        });
    });

    const shareSub = addShareListener((files) => receiveSharedFiles(files));

    // Send anything shared while unpaired as soon as pairing succeeds.
    const unsubscribe = useSessionStore.subscribe((state, prev) => {
      if (state.hasCredentials && !prev.hasCredentials) flushPendingShares();
    });

    return () => {
      active = false;
      cancelAnimationFrame(handle);
      shareSub.remove();
      unsubscribe();
    };
  }, [hydrated]);
}
