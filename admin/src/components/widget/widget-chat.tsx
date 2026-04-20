"use client";

import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import type { WidgetConfig } from "@/lib/widget-config";
import { buildWidgetWebSocketURL } from "@/lib/widget-config";

type Message = {
  id: string;
  role: "assistant" | "user";
  text: string;
};

type ConnectionState = "connecting" | "online" | "offline";

function makeID() {
  if (globalThis.crypto?.randomUUID) {
    return globalThis.crypto.randomUUID();
  }
  return Math.random().toString(36).slice(2);
}

function getVisitorID(tenant: string) {
  const storageKey = `pai-widget:${tenant}:visitor-id`;
  const existing = window.localStorage.getItem(storageKey);
  if (existing) return existing;

  const visitorID = `widget:${tenant}:${makeID()}`;
  window.localStorage.setItem(storageKey, visitorID);
  return visitorID;
}

function parseSocketMessage(data: string): { type?: string; text?: string } {
  try {
    const parsed = JSON.parse(data) as { type?: string; text?: string };
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch {
    return {};
  }
}

export function WidgetChat({ title, tenant, apiOrigin }: WidgetConfig) {
  const socketRef = useRef<WebSocket | null>(null);
  const endRef = useRef<HTMLDivElement | null>(null);
  const [connection, setConnection] = useState<ConnectionState>(() =>
    typeof window !== "undefined" && !("WebSocket" in window) ? "offline" : "connecting",
  );
  const [isTyping, setIsTyping] = useState(false);
  const [draft, setDraft] = useState("");
  const [messages, setMessages] = useState<Message[]>([
    {
      id: "welcome",
      role: "assistant",
      text: "Hi, I can help with quick study questions. What are you working on?",
    },
  ]);

  const websocketURL = useMemo(() => {
    if (typeof window === "undefined") return "";
    return buildWidgetWebSocketURL({ apiOrigin, locationOrigin: window.location.origin });
  }, [apiOrigin]);

  useEffect(() => {
    window.parent.postMessage({ type: "pai-widget:ready", tenant }, "*");
  }, [tenant]);

  useEffect(() => {
    if (!websocketURL || !("WebSocket" in window)) {
      return;
    }

    const socket = new WebSocket(websocketURL);
    socketRef.current = socket;

    socket.addEventListener("open", () => {
      socket.send(JSON.stringify({ type: "auth", user_id: getVisitorID(tenant) }));
    });

    socket.addEventListener("message", (event) => {
      const payload = parseSocketMessage(String(event.data));

      if (payload.type === "auth_ok") {
        setConnection("online");
        return;
      }

      if (payload.type === "typing") {
        setIsTyping(true);
        return;
      }

      if ((payload.type === "response" || payload.type === "notification") && payload.text) {
        setIsTyping(false);
        setMessages((current) => [
          ...current,
          { id: makeID(), role: "assistant", text: payload.text ?? "" },
        ]);
      }
    });

    socket.addEventListener("close", () => {
      setConnection("offline");
      setIsTyping(false);
    });

    socket.addEventListener("error", () => {
      setConnection("offline");
      setIsTyping(false);
    });

    return () => {
      socketRef.current = null;
      socket.close();
    };
  }, [tenant, websocketURL]);

  useEffect(() => {
    endRef.current?.scrollIntoView({ block: "end" });
  }, [messages, isTyping]);

  function requestClose() {
    window.parent.postMessage({ type: "pai-widget:close", tenant }, "*");
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const text = draft.trim();
    const socket = socketRef.current;

    if (!text || !socket || socket.readyState !== WebSocket.OPEN || connection !== "online") {
      return;
    }

    setDraft("");
    setMessages((current) => [...current, { id: makeID(), role: "user", text }]);
    socket.send(JSON.stringify({ type: "message", text }));
  }

  return (
    <section className="flex h-screen min-h-0 flex-col overflow-hidden bg-background text-foreground">
      <header className="flex min-h-16 items-center justify-between border-b bg-card px-4">
        <div className="min-w-0">
          <h1 className="truncate text-base font-semibold">{title}</h1>
          <p className="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
            <span
              aria-hidden="true"
              className={`size-2 rounded-full ${
                connection === "online"
                  ? "bg-emerald-500"
                  : connection === "connecting"
                    ? "bg-amber-500"
                    : "bg-muted-foreground"
              }`}
            />
            {connection === "online" ? "Online" : connection === "connecting" ? "Connecting" : "Offline"}
          </p>
        </div>
        <button
          aria-label="Close chat"
          className="grid size-11 shrink-0 place-items-center rounded-md border bg-background text-muted-foreground transition-colors hover:text-foreground"
          type="button"
          onClick={requestClose}
        >
          <svg aria-hidden="true" className="size-4" viewBox="0 0 16 16" fill="none">
            <path d="M4 4l8 8M12 4l-8 8" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
          </svg>
        </button>
      </header>

      <div className="scrollbar-thin-subtle flex-1 space-y-3 overflow-y-auto bg-muted/30 px-4 py-4">
        {messages.map((message) => (
          <div key={message.id} className={`flex ${message.role === "user" ? "justify-end" : "justify-start"}`}>
            <p
              className={`max-w-[82%] rounded-lg px-3 py-2 text-sm leading-6 ${
                message.role === "user"
                  ? "bg-primary text-primary-foreground"
                  : "border bg-card text-card-foreground"
              }`}
            >
              {message.text}
            </p>
          </div>
        ))}
        {isTyping ? <p className="text-xs text-muted-foreground">Typing...</p> : null}
        <div ref={endRef} />
      </div>

      <form className="flex min-h-18 items-center gap-2 border-t bg-card p-3" onSubmit={handleSubmit}>
        <label className="sr-only" htmlFor="pai-widget-message">
          Message
        </label>
        <input
          id="pai-widget-message"
          className="min-h-11 min-w-0 flex-1 rounded-md border bg-background px-3 text-sm outline-none transition-shadow placeholder:text-muted-foreground focus-visible:ring-2 focus-visible:ring-ring"
          placeholder={connection === "online" ? "Ask a question" : "Connecting..."}
          value={draft}
          disabled={connection !== "online"}
          onChange={(event) => setDraft(event.target.value)}
        />
        <button
          aria-label="Send message"
          className="grid size-11 shrink-0 place-items-center rounded-md bg-primary text-primary-foreground transition-opacity disabled:cursor-not-allowed disabled:opacity-50"
          type="submit"
          disabled={connection !== "online" || draft.trim().length === 0}
        >
          <svg aria-hidden="true" className="size-4" viewBox="0 0 16 16" fill="none">
            <path d="M2 8h11M9 4l4 4-4 4" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </button>
      </form>
    </section>
  );
}
