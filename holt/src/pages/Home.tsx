import { useCallback, useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Chip from '@mui/material/Chip';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import CircularProgress from '@mui/material/CircularProgress';
import Alert from '@mui/material/Alert';
import Collapse from '@mui/material/Collapse';
import Divider from '@mui/material/Divider';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import PlayArrowOutlinedIcon from '@mui/icons-material/PlayArrowOutlined';
import StopOutlinedIcon from '@mui/icons-material/StopOutlined';
import ReplayOutlinedIcon from '@mui/icons-material/ReplayOutlined';
import PauseOutlinedIcon from '@mui/icons-material/PauseOutlined';
import AddCircleOutlinedIcon from '@mui/icons-material/AddCircleOutlined';
import LayersOutlinedIcon from '@mui/icons-material/LayersOutlined';
import { Link } from 'react-router-dom';
import type { DataContainer } from '../types';
import PageHeader from '../components/PageHeader';
import { useNotifications } from '../notifications';
import useOtterEvents from '../useOtterEvents';

type Action = 'start' | 'stop' | 'restart' | 'pause';

interface HomeProps {
  search?: string;
}

function statusDotColor(status: string): 'success' | 'warning' | 'error' | 'default' {
  switch (status) {
    case 'running':
      return 'success';
    case 'paused':
      return 'warning';
    case 'stopped':
    case 'exited':
      return 'error';
    default:
      return 'default';
  }
}

export default function Home({ search = '' }: HomeProps) {
  const { notify } = useNotifications();
  const [all, setAll] = useState<DataContainer[]>([]);
  const [loading, setLoading] = useState(true);
  const [failed, setFailed] = useState(false);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [busyAction, setBusyAction] = useState<Action | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setFailed(false);
    try {
      const res = await window.otter.run('otter', ['list', '--json']);
      if (res.stderr && res.stderr.trim()) {
        notify('warning', 'otter reported', res.stderr.trim());
      }
      setAll(JSON.parse(res.stdout) as DataContainer[]);
    } catch (err) {
      const e = err as { message?: string; stderr?: string };
      setFailed(true);
      notify('error', 'Failed to list containers', e.stderr || e.message || 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, [notify]);

  useEffect(() => {
    void load();
  }, [load]);

  useOtterEvents(() => {
    void load();
  });

  const runAction = async (action: Action, container: DataContainer) => {
    setBusyAction(action);
    try {
      const res = await window.otter.run('otter', [action, container.name]);
      if (res.stderr && res.stderr.trim()) {
        notify('warning', `otter reported on ${action} ${container.name}`, res.stderr.trim());
      }
    } catch (err) {
      const e = err as { message?: string; stderr?: string };
      notify('error', `Failed to ${action} ${container.name}`, e.stderr || e.message || 'Unknown error');
    } finally {
      setBusyAction(null);
    }
  };

  const q = search.trim().toLowerCase();
  const filtered = all.filter(
    (c) =>
      !q ||
      c.name.toLowerCase().includes(q) ||
      c.image.toLowerCase().includes(q),
  );

  const actionsFor = (c: DataContainer): { action: Action; label: string; icon: ReactNode; disabled: boolean }[] => {
    const running = c.status === 'running';
    const paused = c.status === 'paused';
    return [
      { action: 'start', label: 'Start', icon: <PlayArrowOutlinedIcon />, disabled: running || paused },
      { action: 'stop', label: 'Stop', icon: <StopOutlinedIcon />, disabled: !running && !paused },
      { action: 'restart', label: 'Restart', icon: <ReplayOutlinedIcon />, disabled: !running && !paused },
      { action: 'pause', label: 'Pause', icon: <PauseOutlinedIcon />, disabled: !running },
    ];
  };

  const renderRow = (c: DataContainer) => {
    const isExpanded = expanded === c.name;
    const actions = actionsFor(c);
    const dotColor = statusDotColor(c.status);
    return (
      <Card key={c.name} variant="outlined" sx={{ mb: 1.5 }}>
        <CardContent
          sx={{ py: 1.5, cursor: 'pointer' }}
          onClick={() => setExpanded(isExpanded ? null : c.name)}
        >
          <Stack sx={{ flexDirection: 'row', alignItems: 'center', gap: 1.5 }}>
            <Box
              sx={{
                width: 10,
                height: 10,
                borderRadius: '50%',
                flexShrink: 0,
                bgcolor:
                  dotColor === 'success'
                    ? 'success.main'
                    : dotColor === 'warning'
                      ? 'warning.main'
                      : dotColor === 'error'
                        ? 'error.main'
                        : 'grey.500',
              }}
            />
            <Typography variant="h6" component="h2" sx={{ fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {c.name}
            </Typography>
            <Chip label={c.status} color={dotColor} variant="outlined" size="small" />
            <ExpandMoreIcon
              sx={{
                ml: 'auto',
                flexShrink: 0,
                transition: 'transform 150ms ease',
                transform: isExpanded ? 'rotate(180deg)' : 'none',
                color: 'text.secondary',
              }}
            />
          </Stack>
        </CardContent>
        <Collapse in={isExpanded} unmountOnExit>
          <Divider />
          <Box sx={{ py: 1.5, px: 2 }}>
            <Stack sx={{ flexDirection: 'row', alignItems: 'center', gap: 1.5 }}>
              {actions.map((a) => (
                <Tooltip key={a.action} title={a.label}>
                  <span>
                    <IconButton
                      aria-label={a.label}
                      disabled={a.disabled || busyAction !== null}
                      onClick={(e) => {
                        e.stopPropagation();
                        void runAction(a.action, c);
                      }}
                    >
                      {busyAction === a.action ? <CircularProgress size={24} /> : a.icon}
                    </IconButton>
                  </span>
                </Tooltip>
              ))}
            </Stack>
          </Box>
        </Collapse>
      </Card>
    );
  };

  return (
    <Box>
      <PageHeader
        title="Containers"
        subtitle="Manage otter containers"
        loading={loading}
        onRefresh={() => void load()}
      />

      {!loading && !failed && all.length === 0 && (
        <Card variant="outlined" sx={{ textAlign: 'center', py: 6, px: 3 }}>
          <Typography variant="h6" sx={{ fontWeight: 600 }}>
            No containers yet
          </Typography>
          <Typography color="text.secondary" sx={{ mt: 1 }}>
            Create your first otter container to get started.
          </Typography>
          <Stack sx={{ flexDirection: 'row', justifyContent: 'center', gap: 1.5, mt: 3 }}>
            <Button
              component={Link}
              to="/create"
              variant="contained"
              startIcon={<AddCircleOutlinedIcon />}
            >
              Create container
            </Button>
            <Button
              component={Link}
              to="/registry"
              variant="outlined"
              startIcon={<LayersOutlinedIcon />}
            >
              Browse registry
            </Button>
          </Stack>
        </Card>
      )}

      {!loading && all.length > 0 && filtered.length === 0 && q && (
        <Alert severity="info">No containers match &ldquo;{search.trim()}&rdquo;.</Alert>
      )}

      {!loading && all.length > 0 && (
        <Box>{filtered.map(renderRow)}</Box>
      )}
    </Box>
  );
}