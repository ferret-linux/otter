export const DISTRO_NAMES: Record<string, string> = {
  alma: 'AlmaLinux',
  'alma-nonfree': 'AlmaLinux (nonfree)',
  'alma-negativo17': 'AlmaLinux (negativo17)',
  arch: 'ArchLinux',
  guix: 'Guix',
  kali: 'Kali',
  'kali-edge': 'Kali Edge',
  'kali-nonfree': 'Kali (nonfree)',
  'kali-edge-nonfree': 'Kali Edge (nonfree)',
  rhel: 'RHEL',
  artix: 'Artix',
  rocky: 'Rocky Linux',
  'rocky-nonfree': 'Rocky Linux (nonfree)',
  'rocky-negativo17': 'Rocky Linux (negativo17)',
  wolfi: 'Wolfi',
  alpine: 'Alpine',
  'alpine-edge': 'Alpine Edge',
  centos: 'CentOS',
  'centos-nonfree': 'CentOS (nonfree)',
  'centos-negativo17': 'CentOS (negativo17)',
  debian: 'Debian',
  'debian-testing': 'Debian Testing',
  'debian-unstable': 'Debian Unstable',
  'debian-nonfree': 'Debian (nonfree)',
  'debian-testing-nonfree': 'Debian Testing (nonfree)',
  'debian-unstable-nonfree': 'Debian Unstable (nonfree)',
  devuan: 'Devuan',
  'devuan-testing': 'Devuan Testing',
  'devuan-unstable': 'Devuan Unstable',
  'devuan-nonfree': 'Devuan (nonfree)',
  'devuan-testing-nonfree': 'Devuan Testing (nonfree)',
  'devuan-unstable-nonfree': 'Devuan Unstable (nonfree)',
  fedora: 'Fedora',
  'fedora-rawhide': 'Fedora Rawhide',
  'fedora-nonfree': 'Fedora (nonfree)',
  'fedora-rawhide-nonfree': 'Fedora Rawhide (nonfree)',
  'fedora-negativo17': 'Fedora (negativo17)',
  gentoo: 'Gentoo',
  oracle: 'Oracle Linux',
  'oracle-nonfree': 'Oracle Linux (nonfree)',
  'oracle-negativo17': 'Oracle Linux (negativo17)',
  ubuntu: 'Ubuntu',
  'ubuntu-lts': 'Ubuntu LTS',
  'ubuntu-nonfree': 'Ubuntu (nonfree)',
  'ubuntu-lts-nonfree': 'Ubuntu LTS (nonfree)',
  chimera: 'Chimera',
  steamos: 'SteamOS',
  homebrew: 'Homebrew',
  blackarch: 'BlackArch',
  slackware: 'Slackware',
  'slackware-current': 'Slackware Current',
  'void-musl': 'Void (musl)',
  'void-glibc': 'Void (glibc)',
  'void-glibc-nonfree': 'Void (glibc, nonfree)',
  'void-musl-nonfree': 'Void (musl, nonfree)',
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

export function imageBase(ref: string): string {
  return ref.split('/').pop() ?? ref;
}