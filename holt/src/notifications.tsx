import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import { keyframes } from '@emotion/react';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import Badge from '@mui/material/Badge';
import Divider from '@mui/material/Divider';
import ListItem from '@mui/material/ListItem';
import ListItemButton from '@mui/material/ListItemButton';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import NotificationsNoneOutlinedIcon from '@mui/icons-material/NotificationsNoneOutlined';
import CloseOutlinedIcon from '@mui/icons-material/CloseOutlined';
import DeleteSweepOutlinedIcon from '@mui/icons-material/DeleteSweepOutlined';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import CheckCircleOutlinedIcon from '@mui/icons-material/CheckCircleOutlined';
import WarningAmberOutlinedIcon from '@mui/icons-material/WarningAmberOutlined';
import ErrorOutlineOutlinedIcon from '@mui/icons-material/ErrorOutlineOutlined';

export type NotifKind = 'info' | 'success' | 'warning' | 'error';

export interface OtterNotif {
  id: number;
  kind: NotifKind;
  title: string;
  message?: string;
  ts: number;
  read: boolean;
}

const TOAST_MS = 3000;
const FADE_MS = 220;
const MAX_TOASTS = 5;

const shrink = keyframes`
  from { width: 100%; }
  to { width: 0%; }
`;

const growIn = keyframes`
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
`;

function kindColor(kind: NotifKind): string {
  switch (kind) {
    case 'info':
      return 'primary.main';
    case 'success':
      return 'success.main';
    case 'warning':
      return 'warning.main';
    case 'error':
      return 'error.main';
  }
}

function kindIcon(kind: NotifKind): ReactNode {
  const sx = { fontSize: 20 };
  switch (kind) {
    case 'info':
      return <InfoOutlinedIcon sx={{ ...sx, color: 'primary.main' }} />;
    case 'success':
      return <CheckCircleOutlinedIcon sx={{ ...sx, color: 'success.main' }} />;
    case 'warning':
      return <WarningAmberOutlinedIcon sx={{ ...sx, color: 'warning.main' }} />;
    case 'error':
      return <ErrorOutlineOutlinedIcon sx={{ ...sx, color: 'error.main' }} />;
  }
}

function formatTime(ts: number): string {
  return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

interface NotifContextValue {
  notify: (kind: NotifKind, title: string, message?: string) => void;
  markRead: (id: number) => void;
  remove: (id: number) => void;
  clearAll: () => void;
  notifs: OtterNotif[];
  unread: number;
  panelOpen: boolean;
  setPanelOpen: (open: boolean) => void;
}

const NotificationContext = createContext<NotifContextValue | null>(null);

export function useNotifications(): NotifContextValue {
  const ctx = useContext(NotificationContext);
  if (!ctx) throw new Error('useNotifications must be used within NotificationProvider');
  return ctx;
}

let notifId = 1;

export function NotificationProvider({ children }: { children: ReactNode }) {
  const [notifs, setNotifs] = useState<OtterNotif[]>([]);
  const [toasts, setToasts] = useState<OtterNotif[]>([]);
  const [panelOpen, setPanelOpen] = useState(false);

  const notify = useCallback((kind: NotifKind, title: string, message?: string) => {
    const n: OtterNotif = { id: notifId++, kind, title, message, ts: Date.now(), read: false };
    setNotifs((prev) => [n, ...prev]);
    setToasts((prev) => [...prev, n]);
  }, []);

  const markRead = useCallback((id: number) => {
    setNotifs((prev) => prev.map((n) => (n.id === id ? { ...n, read: true } : n)));
  }, []);

  const remove = useCallback((id: number) => {
    setNotifs((prev) => prev.filter((n) => n.id !== id));
    setToasts((prev) => prev.filter((n) => n.id !== id));
  }, []);

  const clearAll = useCallback(() => {
    setNotifs([]);
    setToasts([]);
  }, []);

  const dismissToast = useCallback((id: number) => {
    setToasts((prev) => prev.filter((n) => n.id !== id));
  }, []);

  const unread = useMemo(() => notifs.filter((n) => !n.read).length, [notifs]);

  const value = useMemo(
    () => ({ notify, markRead, remove, clearAll, notifs, unread, panelOpen, setPanelOpen }),
    [notify, markRead, remove, clearAll, notifs, unread, panelOpen],
  );

  const visibleToasts = toasts.slice(-MAX_TOASTS);

  return (
    <NotificationContext.Provider value={value}>
      {children}

      <Stack
        sx={{
          position: 'fixed',
          right: 16,
          bottom: 16,
          zIndex: 2000,
          gap: 1,
          width: 340,
          maxWidth: 'calc(100vw - 32px)',
          pointerEvents: 'none',
        }}
      >
        {visibleToasts.map((n) => (
          <Box key={n.id} sx={{ pointerEvents: 'auto' }}>
            <Toast
              notif={n}
              onDone={dismissToast}
              onRead={() => {
                markRead(n.id);
                dismissToast(n.id);
              }}
            />
          </Box>
        ))}
      </Stack>

      {panelOpen && <NotificationPanel />}
    </NotificationContext.Provider>
  );
}

function Toast({
  notif,
  onDone,
  onRead,
}: {
  notif: OtterNotif;
  onDone: (id: number) => void;
  onRead: () => void;
}) {
  const [closing, setClosing] = useState(false);

  useEffect(() => {
    const t = window.setTimeout(() => setClosing(true), TOAST_MS);
    return () => window.clearTimeout(t);
  }, []);

  useEffect(() => {
    if (!closing) return;
    const t = window.setTimeout(() => onDone(notif.id), FADE_MS);
    return () => window.clearTimeout(t);
  }, [closing, notif.id, onDone]);

  return (
    <Paper
      elevation={4}
      onClick={onRead}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') onRead();
      }}
      sx={{
        cursor: 'pointer',
        p: 1.5,
        borderLeft: 4,
        borderLeftColor: kindColor(notif.kind),
        borderRadius: 2,
        animation: `${growIn} 180ms ease`,
        opacity: closing ? 0 : 1,
        transition: `opacity ${FADE_MS}ms ease`,
      }}
    >
      <Stack direction="row" spacing={1} sx={{ alignItems: 'flex-start' }}>
        {kindIcon(notif.kind)}
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Typography variant="body2" sx={{ fontWeight: 600 }}>
            {notif.title}
          </Typography>
          {notif.message && (
            <Typography
              variant="body2"
              color="text.secondary"
              sx={{ mt: 0.25, overflowWrap: 'anywhere' }}
            >
              {notif.message}
            </Typography>
          )}
        </Box>
      </Stack>
      <Box
        sx={{
          height: 3,
          mt: 1,
          borderRadius: 1,
          bgcolor: 'secondary.main',
          opacity: 0.7,
          animation: `${shrink} ${TOAST_MS}ms linear forwards`,
        }}
      />
    </Paper>
  );
}

export function NotificationToggle() {
  const { unread, panelOpen, setPanelOpen } = useNotifications();
  return (
    <ListItem disablePadding>
      <ListItemButton
        aria-label="Notifications"
        onClick={() => setPanelOpen(!panelOpen)}
        selected={panelOpen}
        sx={{
          '&.Mui-selected': (theme) => ({
            backgroundColor: theme.palette.primary.main + '22',
            color: theme.palette.primary.main,
          }),
        }}
      >
        <ListItemIcon>
          <Badge badgeContent={unread} color="primary" max={99}>
            <NotificationsNoneOutlinedIcon />
          </Badge>
        </ListItemIcon>
        <ListItemText primary="Notifications" />
      </ListItemButton>
    </ListItem>
  );
}

function NotificationPanel() {
  const { notifs, markRead, remove, clearAll, setPanelOpen } = useNotifications();
  return (
    <Paper
      elevation={8}
      sx={{
        position: 'fixed',
        left: 248,
        bottom: 16,
        zIndex: 2000,
        width: 360,
        maxWidth: 'calc(100vw - 264px)',
        maxHeight: 'min(480px, calc(100vh - 120px))',
        display: 'flex',
        flexDirection: 'column',
        borderRadius: 2,
        overflow: 'hidden',
      }}
    >
      <Stack direction="row" sx={{ alignItems: 'center', px: 2, py: 1.25, gap: 1 }}>
        <Typography variant="subtitle1" sx={{ fontWeight: 700 }}>
          Notifications
        </Typography>
        {notifs.length > 0 && (
          <Typography variant="caption" color="text.secondary">
            {notifs.filter((n) => !n.read).length} unread
          </Typography>
        )}
        <Box sx={{ flex: 1 }} />
        <Tooltip title="Clear all">
          <span>
            <IconButton
              size="small"
              aria-label="Clear all notifications"
              onClick={clearAll}
              disabled={notifs.length === 0}
            >
              <DeleteSweepOutlinedIcon fontSize="small" />
            </IconButton>
          </span>
        </Tooltip>
        <IconButton aria-label="Close notifications" size="small" onClick={() => setPanelOpen(false)}>
          <CloseOutlinedIcon fontSize="small" />
        </IconButton>
      </Stack>
      <Divider />
      <Box sx={{ flex: 1, overflow: 'auto', minHeight: 60 }}>
        {notifs.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ textAlign: 'center', py: 4, px: 2 }}>
            No notifications yet.
          </Typography>
        ) : (
          notifs.map((n) => (
            <Stack
              key={n.id}
              direction="row"
              spacing={1.5}
              sx={{
                px: 2,
                py: 1.25,
                alignItems: 'flex-start',
                cursor: 'pointer',
                ...(n.read ? {} : { bgcolor: 'action.hover' }),
                '&:hover': { bgcolor: 'action.selected' },
              }}
              onClick={() => markRead(n.id)}
            >
              {kindIcon(n.kind)}
              <Box sx={{ flex: 1, minWidth: 0 }}>
                <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                  <Typography variant="body2" sx={{ fontWeight: n.read ? 400 : 700 }}>
                    {n.title}
                  </Typography>
                  {!n.read && (
                    <Box sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: 'primary.main', flexShrink: 0 }} />
                  )}
                  <Typography variant="caption" color="text.secondary" sx={{ ml: 'auto', flexShrink: 0 }}>
                    {formatTime(n.ts)}
                  </Typography>
                </Stack>
                {n.message && (
                  <Typography variant="body2" color="text.secondary" sx={{ overflowWrap: 'anywhere' }}>
                    {n.message}
                  </Typography>
                )}
              </Box>
              <IconButton
                size="small"
                aria-label={`Remove notification: ${n.title}`}
                onClick={(e) => {
                  e.stopPropagation();
                  remove(n.id);
                }}
              >
                <CloseOutlinedIcon fontSize="small" />
              </IconButton>
            </Stack>
          ))
        )}
      </Box>
    </Paper>
  );
}