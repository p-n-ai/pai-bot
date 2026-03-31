"use client";

import { useEffect, useEffectEvent, useState } from "react";

export function useAsyncResource<T>(load: () => Promise<T>, deps: readonly unknown[]) {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const runLoad = useEffectEvent(load);

  useEffect(() => {
    let active = true;

    queueMicrotask(() => {
      if (!active) return;
      setLoading(true);
      setError("");
    });

    runLoad()
      .then((result) => {
        if (!active) return;
        setData(result);
      })
      .catch((err) => {
        if (!active) return;
        setData(null);
        setError(err instanceof Error ? err.message : "Failed to load data.");
      })
      .finally(() => {
        if (!active) return;
        setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [...deps]);

  return { data, loading, error, setData, setError };
}
