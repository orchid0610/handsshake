import type { SimpleIcon } from "simple-icons";
import {
  siUbuntu,
  siDebian,
  siArchlinux,
  siFedora,
  siCentos,
  siRedhat,
  siAlpinelinux,
  siOpensuse,
  siLinuxmint,
  siRaspberrypi,
  siKalilinux,
  siRockylinux,
  siAlmalinux,
  siNixos,
  siGentoo,
  siManjaro,
  siFreebsd,
  siOpenbsd,
  siApple,
  siLinux,
} from "simple-icons";

// Keys are the ID= values found in /etc/os-release (or our uname fallbacks).
const BY_ID: Record<string, SimpleIcon> = {
  ubuntu: siUbuntu,
  debian: siDebian,
  arch: siArchlinux,
  fedora: siFedora,
  centos: siCentos,
  rhel: siRedhat,
  alpine: siAlpinelinux,
  opensuse: siOpensuse,
  "opensuse-leap": siOpensuse,
  "opensuse-tumbleweed": siOpensuse,
  suse: siOpensuse,
  linuxmint: siLinuxmint,
  raspbian: siRaspberrypi,
  kali: siKalilinux,
  rocky: siRockylinux,
  almalinux: siAlmalinux,
  nixos: siNixos,
  gentoo: siGentoo,
  manjaro: siManjaro,
  freebsd: siFreebsd,
  openbsd: siOpenbsd,
  macos: siApple,
  linux: siLinux,
};

/** Look up the logo for an OS id; fall back to its relatives (ID_LIKE), then to generic Linux. */
export function iconFor(id: string, like = ""): SimpleIcon {
  for (const key of [id, ...like.split(/\s+/)]) {
    if (key && BY_ID[key]) return BY_ID[key];
  }
  return siLinux;
}

/** Dark glyph on light brand colours, white glyph otherwise. */
export function glyphFor(hex: string): string {
  const n = parseInt(hex, 16);
  const lum =
    (0.299 * (n >> 16) + 0.587 * ((n >> 8) & 255) + 0.114 * (n & 255)) / 255;
  return lum > 0.62 ? "#0d0f14" : "#ffffff";
}
