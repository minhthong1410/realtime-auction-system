"use client";

import { useEffect, useRef, useCallback, useId } from "react";
import { useAuthStore } from "@/stores/auth-store";
import type { WSMessage, WSNewBid, WSAuctionEnded, WSBalanceUpdate } from "@/lib/types";

const WS_URL = process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080";

type MessageHandler = (msg: WSMessage) => void;

let ws: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
const subscribers = new Map<string, Set<MessageHandler>>();
const subscribedRooms = new Set<string>();

export function reconnectWS() {
  if (ws) {
    ws.onclose = null; // prevent auto-reconnect with old token
    ws.close();
    ws = null;
  }
  connect();
}

function connect() {
  if (ws?.readyState === WebSocket.OPEN || ws?.readyState === WebSocket.CONNECTING) return;

  const token = typeof window !== "undefined" ? localStorage.getItem("access_token") : null;
  const url = token ? `${WS_URL}/ws?token=${token}` : `${WS_URL}/ws`;

  ws = new WebSocket(url);

  ws.onopen = () => {
    subscribedRooms.forEach((room) => {
      ws?.send(JSON.stringify({ action: "subscribe", room }));
    });
  };

  ws.onmessage = (event) => {
    try {
      const msg: WSMessage = JSON.parse(event.data);
      subscribers.forEach((handlers) => {
        handlers.forEach((handler) => handler(msg));
      });
    } catch {
      // ignore parse errors
    }
  };

  ws.onclose = () => {
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
  const subId = useId();

  useEffect(() => {
    handlerRef.current = onMessage;
  });

  useEffect(() => {
    connect();

    const handler: MessageHandler = (msg) => {
      handlerRef.current?.(msg);
    };

    if (!subscribers.has(subId)) {
      subscribers.set(subId, new Set());
    }
    subscribers.get(subId)!.add(handler);

    rooms.forEach(subscribeRoom);

    return () => {
      subscribers.get(subId)?.delete(handler);
      if (subscribers.get(subId)?.size === 0) {
        subscribers.delete(subId);
      }
      rooms.forEach((room) => {
        let needed = false;
        subscribers.forEach((handlers) => {
          if (handlers.size > 0) needed = true;
        });
        if (!needed) unsubscribeRoom(room);
      });
    };
  }, [rooms.join(","), subId]); // eslint-disable-line react-hooks/exhaustive-deps
}

export function useBalanceSync() {
  const { user, setBalance } = useAuthStore();

  const rooms = user ? [`user:${user.id}`] : [];

  useWebSocket(rooms, useCallback((msg: WSMessage) => {
    if (msg.type === "balance_update") {
      const data = msg.data as WSBalanceUpdate;
      setBalance(data.balance);
    }
  }, [setBalance]));
}

export type { WSNewBid, WSAuctionEnded, WSBalanceUpdate };
