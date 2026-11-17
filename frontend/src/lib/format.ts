/** "12 minutes ago", "Yesterday", "Nov 3": for the recent-sessions list. */
export function ago(unixSeconds: number, nowMs = Date.now()): string {
  const s = Math.max(0, Math.floor(nowMs / 1000) - unixSeconds);
  if (s < 45) return "Just now";
  const m = Math.max(1, Math.floor(s / 60));
  if (m < 60) return `${m} minute${m === 1 ? "" : "s"} ago`;
  const h = Math.floor(s / 3600);
  if (h < 24) return `${h} hour${h === 1 ? "" : "s"} ago`;
  const d = Math.floor(s / 86400);
  if (d === 1) return "Yesterday";
  if (d < 7) return `${d} days ago`;
  return new Date(unixSeconds * 1000).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  });
}

/** "ssh-ed25519" becomes "ED25519", and so on, for the tab bar. */
export function keyLabel(t: string): string {
  if (t.startsWith("ssh-ed25519")) return "ED25519";
  if (t.startsWith("ssh-rsa") || t.startsWith("rsa-sha2")) return "RSA";
  if (t.startsWith("ecdsa-")) return "ECDSA";
  if (t.startsWith("ssh-dss")) return "DSA";
  return t;
}
