"use client";

import { useState, useTransition } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { AdminSurface, AdminSurfaceHeader } from "@/components/admin-surface";
import { StatePanel } from "@/components/state-panel";
import { getWhatsAppStatus, disconnectWhatsApp } from "@/lib/api";

export function WhatsAppSetupPanel() {
  const queryClient = useQueryClient();
  const [isPending, startTransition] = useTransition();
  const [disconnectError, setDisconnectError] = useState("");

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["whatsapp", "status"],
    queryFn: getWhatsAppStatus,
    refetchInterval: 5000,
  });

  function handleDisconnect() {
    setDisconnectError("");
    startTransition(async () => {
      try {
        await disconnectWhatsApp();
        queryClient.invalidateQueries({ queryKey: ["whatsapp", "status"] });
      } catch (err) {
        setDisconnectError(
          err instanceof Error ? err.message : "Failed to disconnect",
        );
      }
    });
  }

  if (isLoading) {
    return <StatePanel tone="loading" title="Loading WhatsApp status..." description="Checking connection to WhatsApp server." />;
  }

  if (error) {
    return (
      <StatePanel
        tone="error"
        title="Could not load WhatsApp status"
        description={
          error instanceof Error
            ? error.message
            : "Check that WhatsApp is enabled on the server."
        }
      />
    );
  }

  if (data?.connected) {
    return (
      <AdminSurface>
        <AdminSurfaceHeader
          title="WhatsApp connected"
          description="Your WhatsApp account is linked and the bot is active."
        />
        <div className="mt-6 flex items-center justify-between rounded-lg border border-emerald-200 bg-emerald-50 p-4 dark:border-emerald-800 dark:bg-emerald-950/30">
          <div className="flex items-center gap-3">
            <span className="text-2xl">&#9989;</span>
            <div>
              <p className="font-medium text-emerald-900 dark:text-emerald-100">
                Session active
              </p>
              <p className="text-sm text-emerald-700 dark:text-emerald-300">
                Messages are being sent and received via WhatsApp.
              </p>
            </div>
          </div>
          <button
            onClick={handleDisconnect}
            disabled={isPending}
            className="rounded-md border border-red-300 bg-white px-4 py-2 text-sm font-medium text-red-700 hover:bg-red-50 disabled:opacity-50 dark:border-red-700 dark:bg-red-950/30 dark:text-red-300 dark:hover:bg-red-950/50"
          >
            {isPending ? "Disconnecting..." : "Disconnect"}
          </button>
        </div>
        {disconnectError && (
          <p className="mt-2 text-sm text-destructive">{disconnectError}</p>
        )}
      </AdminSurface>
    );
  }

  return (
    <AdminSurface>
      <AdminSurfaceHeader
        title="Link WhatsApp"
        description="Scan the QR code below with your phone to connect."
      />
      <div className="mt-6 space-y-4">
        {data?.qr_image ? (
          <div className="flex flex-col items-center gap-4">
            <img
              src={data.qr_image}
              alt="WhatsApp QR Code"
              className="h-64 w-64 rounded-lg border"
            />
            <div className="text-center text-sm text-muted-foreground">
              <p>
                Open WhatsApp on your phone &rarr; <strong>Settings</strong>{" "}
                &rarr; <strong>Linked Devices</strong> &rarr;{" "}
                <strong>Link a Device</strong>
              </p>
              <p className="mt-1">Page refreshes automatically every 5 seconds.</p>
            </div>
          </div>
        ) : (
          <div className="flex flex-col items-center gap-4 py-8">
            <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-foreground" />
            <p className="text-sm text-muted-foreground">
              Waiting for QR code from server...
            </p>
            <button
              onClick={() => refetch()}
              className="text-sm text-primary underline hover:no-underline"
            >
              Retry
            </button>
          </div>
        )}
      </div>
    </AdminSurface>
  );
}
