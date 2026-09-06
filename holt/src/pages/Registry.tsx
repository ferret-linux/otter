import { useCallback, useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Chip from '@mui/material/Chip';
import Button from '@mui/material/Button';
import Tooltip from '@mui/material/Tooltip';
import CircularProgress from '@mui/material/CircularProgress';
import Alert from '@mui/material/Alert';
import RefreshIcon from '@mui/icons-material/Refresh';
import DownloadIcon from '@mui/icons-material/Download';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorOutlinedIcon from '@mui/icons-material/ErrorOutlined';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import type { DataContainer, RegistryEntry } from '../types';
import { prettyName } from '../distros';
import PageHeader from '../components/PageHeader';

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
    case 'not_pulled':
      return { label: 'Not pulled', color: 'error' };
  }
}

type BusyAction = 'pull' | 'remove';

interface RegistryProps {
  search?: string;
}

export default function Registry({ search }: RegistryProps) {
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
      const warnings = [reg.stderr, containers.stderr].filter(nonEmpty).join('\n');
      if (warnings) setError(warnings);
      setEntries(JSON.parse(reg.stdout) as RegistryEntry[]);
      const list = JSON.parse(containers.stdout) as DataContainer[];
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
      await load();
      if (stderr.trim()) setError(stderr);
    } catch (err) {
      const e = err as { message?: string; stderr?: string };
      setError(e.stderr || e.message || 'Command failed');
    } finally {
      setBusy(null);
    }
  };

  const q = (search ?? '').trim().toLowerCase();
  const filteredEntries = q
    ? entries.filter(
        (entry) =>
          entry.name.toLowerCase().includes(q) ||
          imageBase(entry.image).toLowerCase().includes(q),
      )
    : entries;

  return (
    <Box>
      <PageHeader
        title="Registry"
        subtitle="Container images available for otter"
        loading={loading}
        onRefresh={() => void load()}
      />

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {!loading && !error && entries.length === 0 && (
        <Alert severity="info">No entries in the registry.</Alert>
      )}

      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(400px, 1fr))',
          gap: 2,
        }}
      >
        {filteredEntries.map((entry) => {
          const chip = stalenessChip(entry);
          const inUse = inUseImages.includes(imageBase(entry.image));
          const pulled = entry.pulled;
          const stale = entry.staleness;
          const isBusy = busy?.name === entry.name;

          let syncLabel: string;
          let syncIcon: ReactNode = <DownloadIcon fontSize="small" />;
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

      {!loading && entries.length > 0 && q && filteredEntries.length === 0 && (
        <Alert severity="info">No entries match &ldquo;{search?.trim()}&rdquo;.</Alert>
      )}
    </Box>
  );
}