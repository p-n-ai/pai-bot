"use client";

import { useEffect, useState, type DependencyList } from "react";

export function useAsyncResource<T>(load: () => Promise<T>, deps: DependencyList) {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    setLoading(true);
    setError("");

    load()
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
  }, deps);

  return { data, loading, error, setData, setError };
}
