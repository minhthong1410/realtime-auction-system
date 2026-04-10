"use client";

import { useState, useEffect } from "react";

export function useCountdown(endTime: string) {
  const [timeLeft, setTimeLeft] = useState(() => getTimeLeft(endTime));

  useEffect(() => {
    const timer = setInterval(() => {
      const left = getTimeLeft(endTime);
      setTimeLeft(left);
      if (left.total <= 0) clearInterval(timer);
    }, 1000);

    return () => clearInterval(timer);
  }, [endTime]);

  return timeLeft;
}

function getTimeLeft(endTime: string) {
  const total = Math.max(0, new Date(endTime).getTime() - Date.now());
  const seconds = Math.floor((total / 1000) % 60);
  const minutes = Math.floor((total / 1000 / 60) % 60);
  const hours = Math.floor((total / (1000 * 60 * 60)) % 24);
  const days = Math.floor(total / (1000 * 60 * 60 * 24));

  return { total, days, hours, minutes, seconds };
}

export function formatTimeLeft(t: ReturnType<typeof getTimeLeft>) {
  if (t.total <= 0) return "Ended";
  if (t.days > 0) return `${t.days}d ${t.hours}h ${t.minutes}m`;
  if (t.hours > 0) return `${t.hours}h ${t.minutes}m ${t.seconds}s`;
  return `${t.minutes}m ${t.seconds}s`;
}
