import { useEffect, useRef, useState } from 'react';
import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Typography from '@mui/material/Typography';
import Button from '@mui/material/Button';
import DeleteSweepOutlinedIcon from '@mui/icons-material/DeleteSweepOutlined';
import type { LogEntry } from '../types';

const MONO = 'Menlo, Consolas, "Liberation Mono", monospace';

const formatTime = (ts: number): string =>
  new Date(ts).toLocaleTimeString([], { hour12: false });

const renderStatus = (e: LogEntry) => {
  if (e.status === 'running') {
    return <Box component="span" sx={{ color: 'primary.main' }}>running…</Box>;
  }
  const ok = e.status === 'ok';
  return (
    <Box component="span" sx={{ color: ok ? 'success.main' : 'error.main' }}>
      {ok ? 'ok' : `err ${e.exitCode ?? 'sig'}`} · {e.durationMs}ms
    </Box>
  );
};

export default function Verbose() {
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const [loaded, setLoaded] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    window.otter
      .getLog()
      .then((log) => setEntries(log))
      .catch(() => {})
      .finally(() => setLoaded(true));
    const unsubscribe = window.otter.onLog((entry) => {
      setEntries((prev) => {
        const idx = prev.findIndex((e) => e.id === entry.id);
        if (idx === -1) return [...prev, entry];
        const next = prev.slice();
        next[idx] = entry;
        return next;
      });
    });
    return unsubscribe;
  }, []);

  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 120;
    if (nearBottom) el.scrollTop = el.scrollHeight;
  }, [entries]);

  const clear = async () => {
    await window.otter.clearLog();
    setEntries([]);
  };

  return (
    <Box sx={{ maxWidth: 960 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
        <Typography variant="h5" component="h1" sx={{ fontWeight: 700, flexGrow: 1 }}>
          Verbose
        </Typography>
        <Button
          variant="outlined"
          startIcon={<DeleteSweepOutlinedIcon />}
          disabled={entries.length === 0}
          onClick={() => void clear()}
        >
          Clear
        </Button>
      </Box>

      <Card variant="outlined">
        <CardContent sx={{ p: 0 }}>
          <Box
            ref={scrollRef}
            sx={{
              height: '60vh',
              overflow: 'auto',
              p: 2,
              fontFamily: MONO,
              fontSize: 13,
              lineHeight: 1.6,
              bgcolor: '#141717',
            }}
          >
            {loaded && entries.length === 0 && (
              <Typography variant="body2" sx={{ color: 'text.disabled', fontFamily: MONO }}>
                No otter commands logged yet.
              </Typography>
            )}

            {entries.map((e) => (
              <Box key={e.id} sx={{ mb: 2, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                <Box sx={{ display: 'flex', gap: 1 }}>
                  <Box component="span" sx={{ color: 'primary.main' }}>$</Box>
                  <Box component="span" sx={{ color: 'text.primary' }}>
                    {[e.command, ...e.args].join(' ')}
                  </Box>
                  <Box component="span" sx={{ color: 'text.disabled' }}>
                    {formatTime(e.ts)}
                  </Box>
                </Box>
                {e.input !== undefined && (
                  <Box component="div" sx={{ color: 'text.disabled' }}>
                    │ {e.input}
                  </Box>
                )}
                {e.stdout && (
                  <Box component="div" sx={{ color: 'text.secondary' }}>
                    {e.stdout.trim()}
                  </Box>
                )}
                {e.stderr && (
                  <Box component="div" sx={{ color: 'error.main' }}>
                    {e.stderr.trim()}
                  </Box>
                )}
                <Box component="div" sx={{ color: 'text.disabled' }}>
                  {renderStatus(e)}
                </Box>
              </Box>
            ))}
          </Box>
        </CardContent>
      </Card>
    </Box>
  );
}