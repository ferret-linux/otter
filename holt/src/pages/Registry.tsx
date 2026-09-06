import { useCallback, useEffect, useState } from 'react';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Chip from '@mui/material/Chip';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import CircularProgress from '@mui/material/CircularProgress';
import Alert from '@mui/material/Alert';
import RefreshIcon from '@mui/icons-material/Refresh';
import DownloadIcon from '@mui/icons-material/Download';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorOutlinedIcon from '@mui/icons-material/ErrorOutlined';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import type { RegistryEntry } from '../types';

const DISTRO_NAMES: Record<string, string> = {
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

function prettyName(name: string): string {
  if (DISTRO_NAMES[name]) return DISTRO_NAMES[name];
  return name
    .split('-')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

function imageBase(ref: string): string {
  return ref.split('/').pop() ?? ref;
}

function formatBytes(bytes: number): string {
  if (bytes <= 0) return '—';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.min(Math.floor(Math.log2(bytes) / 10), units.length - 1);
  const value = bytes / 2 ** (10 * i);
  return `${value.toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  if (Number.isNaN(diff)) return iso;
  const mins = Math.round(diff / 60_000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.round(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.round(hours / 24);
  if (days < 30) return `${days}d ago`;
  return new Date(iso).toLocaleDateString();
}

interface StalenessChip {
  label: string;
  color: 'success' | 'warning' | 'info' | 'error' | 'default';
}

function stalenessChip(entry: RegistryEntry): StalenessChip {
  switch (entry.staleness) {
    case 'current':
      return { label: 'Up to date', color: 'success' };
    case 'behind':
      return {
        label: entry.behind_count ? `${entry.behind_count} behind` : 'Behind',
        color: 'warning',
      };
    case 'ahead':
      return { label: 'Ahead', color: 'info' };
    case 'unknown':
      return { label: 'Unknown', color: 'default' };
    default:
      return { label: 'Not pulled', color: 'error' };
  }
}

interface ContainerInfo {
  image: string;
}

type BusyAction = 'pull' | 'remove';

export default function Registry() {
  const [entries, setEntries] = useState<RegistryEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<{ name: string; action: BusyAction } | null>(null);
  const [inUseImages, setInUseImages] = useState<string[]>([]);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [reg, containers] = await Promise.all([
        window.otter.run('otter', ['reg', 'list', '--json']),
        window.otter.run('otter', ['list', '--json']),
      ]);
      const nonEmpty = (s: string) => s.trim().length > 0;
      if (reg.stderr && nonEmpty(reg.stderr)) setError(reg.stderr);
      if (containers.stderr && nonEmpty(containers.stderr)) setError(containers.stderr);
      setEntries(JSON.parse(reg.stdout) as RegistryEntry[]);
      const list = JSON.parse(containers.stdout) as ContainerInfo[];
      setInUseImages(list.map((c) => imageBase(c.image)));
    } catch (err) {
      const e = err as { message?: string; stderr?: string };
      setError(e.stderr || e.message || 'Failed to load registry');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const run = async (name: string, action: BusyAction, args: string[]) => {
    setBusy({ name, action });
    setError(null);
    try {
      const { stderr } = await window.otter.run('otter', args);
      if (stderr) setError(stderr);
      await load();
    } catch (err) {
      const e = err as { message?: string; stderr?: string };
      setError(e.stderr || e.message || 'Command failed');
    } finally {
      setBusy(null);
    }
  };

  return (
    <Box>
      <Stack
        sx={{
          flexDirection: 'row',
          alignItems: 'center',
          justifyContent: 'space-between',
          mb: 2,
        }}
      >
        <Box>
          <Typography variant="h5" component="h1" sx={{ fontWeight: 700 }}>
            Registry
          </Typography>
          <Typography color="text.secondary" variant="body2">
            Container images available for otter
          </Typography>
        </Box>
        <Tooltip title="Refresh">
          <span>
            <IconButton onClick={() => void load()} disabled={loading} aria-label="Refresh list">
              {loading ? <CircularProgress size={20} /> : <RefreshIcon />}
            </IconButton>
          </span>
        </Tooltip>
      </Stack>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {!loading && entries.length === 0 && (
        <Alert severity="info">No entries in the registry.</Alert>
      )}

      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(400px, 1fr))',
          gap: 2,
        }}
      >
        {entries.map((entry) => {
          const chip = stalenessChip(entry);
          const inUse = inUseImages.includes(imageBase(entry.image));
          const pulled = entry.pulled;
          const stale = entry.staleness;
          const isBusy = busy?.name === entry.name;

          let syncLabel: string;
          let syncIcon: React.ReactNode = <DownloadIcon fontSize="small" />;
          let syncDisabled = false;
          let syncTooltip = `Pull ${prettyName(entry.name)}`;
          if (stale === 'behind' && pulled) {
            syncLabel = 'Refresh';
            syncIcon = <RefreshIcon fontSize="small" />;
            syncTooltip = 'Update to the latest build';
          } else if (stale === 'current') {
            syncLabel = 'Up to date';
            syncDisabled = true;
            syncTooltip = 'Image is already up to date';
          } else if (stale === 'ahead') {
            syncLabel = 'Ahead';
            syncDisabled = true;
            syncTooltip = 'Pulled image is ahead of the registry';
          } else {
            syncLabel = 'Pull';
          }

          let statusLabel = chip.label;
          let statusColor = chip.color;
          if (isBusy) {
            if (busy?.action === 'pull') {
              statusLabel = stale === 'behind' && pulled ? 'Refreshing' : 'Pulling';
              statusColor = 'info';
            } else {
              statusLabel = 'Removing';
              statusColor = 'error';
            }
          }

          const removeDisabled = isBusy || !pulled || inUse;
          const removeTooltip = inUse
            ? 'Image is in use by a container'
            : !pulled
              ? 'Image is not pulled'
              : `Remove ${prettyName(entry.name)}`;

          return (
            <Card
              key={entry.name}
              variant="outlined"
              sx={{ display: 'flex', alignItems: 'stretch', minHeight: 168 }}
            >
              <CardContent sx={{ flexGrow: 1, minWidth: 0 }}>
                <Stack direction="row" sx={{ alignItems: 'center', gap: 1 }}>
                  {isBusy ? (
                    <CircularProgress size={20} />
                  ) : chip.color === 'success' ? (
                    <CheckCircleIcon sx={{ color: 'success.main', fontSize: 20 }} />
                  ) : chip.color === 'error' ? (
                    <ErrorOutlinedIcon sx={{ color: 'error.main', fontSize: 20 }} />
                  ) : null}
                  <Typography variant="h6" component="h2" sx={{ fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {prettyName(entry.name)}
                  </Typography>
                  <Chip label={statusLabel} color={statusColor} variant="outlined" size="small" />
                  {inUse && (
                    <Tooltip title="In use by a container">
                      <Chip
                        icon={<LockOutlinedIcon />}
                        label="In use"
                        size="small"
                        variant="outlined"
                        sx={{ ml: 'auto' }}
                      />
                    </Tooltip>
                  )}
                </Stack>

                <Typography
                  variant="body2"
                  component="div"
                  color="text.secondary"
                  sx={{ fontFamily: 'monospace', mt: 0.5 }}
                >
                  {entry.image}
                </Typography>

                <Stack direction="row" spacing={1} useFlexGap sx={{ flexWrap: 'wrap', mt: 1 }}>
                  {entry.architecture.map((arch) => (
                    <Chip key={arch} label={arch} size="small" variant="outlined" />
                  ))}
                </Stack>

                <Stack direction="row" spacing={3} sx={{ mt: 1.5, color: 'text.secondary' }}>
                  <span>{formatBytes(entry.size ?? 0)}</span>
                  <span>Built {relativeTime(entry.built_at)}</span>
                </Stack>
              </CardContent>

              <Stack
                sx={{
                  justifyContent: 'center',
                  px: 1.5,
                  borderLeft: 1,
                  borderColor: 'divider',
                  gap: 1,
                }}
              >
                <Tooltip title={syncTooltip}>
                  <span>
                    <Button
                      size="small"
                      variant="outlined"
                      fullWidth
                      startIcon={syncIcon}
                      disabled={syncDisabled || isBusy}
                      onClick={() => void run(entry.name, 'pull', ['reg', 'pull', entry.name])}
                      sx={{ justifyContent: 'flex-start' }}
                    >
                      {syncLabel}
                    </Button>
                  </span>
                </Tooltip>
                <Tooltip title={removeTooltip}>
                  <span>
                    <Button
                      size="small"
                      variant="outlined"
                      color="error"
                      fullWidth
                      startIcon={<DeleteOutlinedIcon fontSize="small" />}
                      disabled={removeDisabled}
                      onClick={() => void run(entry.name, 'remove', ['reg', 'remove', entry.name])}
                      sx={{ justifyContent: 'flex-start' }}
                    >
                      Remove
                    </Button>
                  </span>
                </Tooltip>
              </Stack>
            </Card>
          );
        })}
      </Box>
    </Box>
  );
}