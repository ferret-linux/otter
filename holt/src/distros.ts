export const DISTRO_NAMES: Record<string, string> = {
  alma: 'AlmaLinux',
  arch: 'ArchLinux',
  guix: 'Guix',
  kali: 'Kali',
  'kali-edge': 'Kali Edge',
  rhel: 'RHEL',
  artix: 'Artix',
  nixos: 'NixOS',
  'nixos-unstable': 'NixOS Unstable',
  rocky: 'Rocky Linux',
  wolfi: 'Wolfi',
  alpine: 'Alpine',
  'alpine-edge': 'Alpine Edge',
  centos: 'CentOS',
  debian: 'Debian',
  'debian-testing': 'Debian Testing',
  'debian-unstable': 'Debian Unstable',
  devuan: 'Devuan',
  'devuan-testing': 'Devuan Testing',
  'devuan-unstable': 'Devuan Unstable',
  fedora: 'Fedora',
  'fedora-rawhide': 'Fedora Rawhide',
  gentoo: 'Gentoo',
  oracle: 'Oracle Linux',
  ubuntu: 'Ubuntu',
  'ubuntu-lts': 'Ubuntu LTS',
  chimera: 'Chimera',
  steamos: 'SteamOS',
  homebrew: 'Homebrew',
  blackarch: 'BlackArch',
  slackware: 'Slackware',
  'slackware-current': 'Slackware Current',
  'void-musl': 'Void (musl)',
  'void-glibc': 'Void (glibc)',
  amazonlinux: 'Amazon Linux',
  'opensuse-leap': 'openSUSE Leap',
  'opensuse-tumbleweed': 'Tumbleweed',
};

export function prettyName(name: string): string {
  if (DISTRO_NAMES[name]) return DISTRO_NAMES[name];
  return name
    .split('-')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}