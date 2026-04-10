"use client";

import { useEffect, useRef, useCallback } from "react";
import { useAuthStore } from "@/stores/auth-store";
import type { WSMessage, WSNewBid, WSAuctionEnded, WSBalanceUpdate } from "@/lib/types";

const WS_URL = process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080";

type MessageHandler = (msg: WSMessage) => void;

let ws: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
const subscribers = new Map<string, Set<MessageHandler>>();
const subscribedRooms = new Set<string>();

function getWS(): WebSocket | null {
  return ws;
}

function connect() {
  if (ws?.readyState === WebSocket.OPEN || ws?.readyState === WebSocket.CONNECTING) return;

  const token = typeof window !== "undefined" ? localStorage.getItem("access_token") : null;
  const url = token ? `${WS_URL}/ws?token=${token}` : `${WS_URL}/ws`;

  ws = new WebSocket(url);

  ws.onopen = () => {
    // Re-subscribe to rooms after reconnect
    subscribedRooms.forEach((room) => {
      ws?.send(JSON.stringify({ action: "subscribe", room }));
    });
  };

  ws.onmessage = (event) => {
    try {
      const msg: WSMessage = JSON.parse(event.data);
      // Broadcast to all subscribers
      subscribers.forEach((handlers) => {
        handlers.forEach((handler) => handler(msg));
      });
    } catch {
      // ignore parse errors
    }
  };

  ws.onclose = () => {
    // Auto reconnect after 3s
    if (reconnectTimer) clearTimeout(reconnectTimer);
    reconnectTimer = setTimeout(connect, 3000);
  };

  ws.onerror = () => {
    ws?.close();
  };
}

function subscribeRoom(room: string) {
  subscribedRooms.add(room);
  if (ws?.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ action: "subscribe", room }));
  }
}

function unsubscribeRoom(room: string) {
  subscribedRooms.delete(room);
  if (ws?.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ action: "unsubscribe", room }));
  }
}

export function useWebSocket(
  rooms: string[],
  onMessage?: MessageHandler
) {
  const handlerRef = useRef(onMessage);
  handlerRef.current = onMessage;

  const id = useRef(Math.random().toString(36).slice(2));

  useEffect(() => {
    connect();

    const handler: MessageHandler = (msg) => {
      handlerRef.current?.(msg);
    };

    const subId = id.current;
    if (!subscribers.has(subId)) {
      subscribers.set(subId, new Set());
    }
    subscribers.get(subId)!.add(handler);

    // Subscribe to rooms
    rooms.forEach(subscribeRoom);

    return () => {
      subscribers.get(subId)?.delete(handler);
      if (subscribers.get(subId)?.size === 0) {
        subscribers.delete(subId);
      }
      // Only unsubscribe rooms if no other subscribers need them
      rooms.forEach((room) => {
        let needed = false;
        subscribers.forEach((handlers) => {
          if (handlers.size > 0) needed = true;
        });
        if (!needed) unsubscribeRoom(room);
      });
    };
  }, [rooms.join(",")]); // eslint-disable-line react-hooks/exhaustive-deps
}

// Hook for auto-syncing user balance via WS
export function useBalanceSync() {
  const { user, setBalance, isAuthenticated } = useAuthStore();

  const rooms = user ? [`user:${user.id}`] : [];

  useWebSocket(rooms, useCallback((msg: WSMessage) => {
    if (msg.type === "balance_update") {
      const data = msg.data as WSBalanceUpdate;
      setBalance(data.balance);
    }
  }, [setBalance]));
}

export type { WSNewBid, WSAuctionEnded, WSBalanceUpdate };
