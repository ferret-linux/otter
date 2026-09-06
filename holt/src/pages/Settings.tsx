import { useEffect, useMemo, useState } from 'react';
import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Typography from '@mui/material/Typography';
import Button from '@mui/material/Button';
import Divider from '@mui/material/Divider';
import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import Switch from '@mui/material/Switch';
import TextField from '@mui/material/TextField';
import CircularProgress from '@mui/material/CircularProgress';
import Alert from '@mui/material/Alert';
import type { SettingsEntry } from '../types';

const isInt = (value: string): boolean => /^-?\d+$/.test(value.trim());

type FieldValue = string | boolean;

export default function Settings() {
  const [entries, setEntries] = useState<SettingsEntry[]>([]);
  const [values, setValues] = useState<Record<string, FieldValue>>({});
  const [initial, setInitial] = useState<Record<string, FieldValue>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<'success' | 'error' | null>(null);
  const [resultMessage, setResultMessage] = useState('');

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const { stdout, stderr } = await window.otter.run('otter', ['settings', '--list-json']);
      if (stderr) setError(stderr);
      const list = JSON.parse(stdout) as SettingsEntry[];
      const vals: Record<string, FieldValue> = {};
      for (const e of list) vals[e.field] = e.value;
      setEntries(list);
      setValues(vals);
      setInitial(vals);
    } catch (err) {
      const e = err as { message?: string; stderr?: string };
      setError(e.stderr || e.message || 'Failed to load settings');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, []);

  const dirty = useMemo(
    () => entries.some((e) => values[e.field] !== initial[e.field]),
    [entries, values, initial],
  );

  const set = (field: string, value: FieldValue) =>
    setValues((v) => ({ ...v, [field]: value }));

  const save = async () => {
    setResult(null);
    setResultMessage('');
    setBusy(true);
    const payload: Record<string, FieldValue> = {};
    for (const e of entries) payload[e.field] = values[e.field];
    try {
      const { stderr } = await window.otter.run(
        'otter',
        ['settings', '--apply-json'],
        JSON.stringify(payload),
      );
      if (stderr) {
        setResult('error');
        setResultMessage(stderr.trim());
      } else {
        setResult('success');
        setResultMessage('Settings saved.');
        setInitial({ ...values });
      }
    } catch (err) {
      const e = err as { message?: string; stderr?: string };
      setResult('error');
      setResultMessage(e.stderr || e.message || 'Failed to save settings');
    } finally {
      setBusy(false);
    }
  };

  const sections = useMemo(() => {
    const out: { section: string; items: SettingsEntry[] }[] = [];
    for (const e of entries) {
      const s = out.find((o) => o.section === e.section);
      if (s) s.items.push(e);
      else out.push({ section: e.section, items: [e] });
    }
    return out;
  }, [entries]);

  const renderField = (e: SettingsEntry) => {
    const value = values[e.field];
    const invalidNumeric =
      e.kind === 'text' && e.numeric && typeof value === 'string' && value.length > 0 && !isInt(value);

    if (e.kind === 'toggle') {
      return (
        <Box key={e.field}>
          <FormControlLabel
            control={
              <Switch
                checked={Boolean(value)}
                onChange={(ev) => set(e.field, ev.target.checked)}
                disabled={busy}
              />
            }
            label={e.field}
          />
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', ml: 4, mb: 2 }}>
            {e.description}
          </Typography>
        </Box>
      );
    }

    if (e.options && e.options.length > 0) {
      return (
        <Box key={e.field} sx={{ mb: 2 }}>
          <FormControl fullWidth>
            <InputLabel id={`${e.field}-label`}>{e.field}</InputLabel>
            <Select
              labelId={`${e.field}-label`}
              value={typeof value === 'string' ? value : ''}
              onChange={(ev) => set(e.field, ev.target.value)}
              label={e.field}
              disabled={busy}
            >
              {e.options.map((opt) => (
                <MenuItem key={opt} value={opt}>
                  {opt}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          <Typography variant="caption" color="text.secondary" sx={{ ml: 1, mt: 0.5 }}>
            {e.description}
          </Typography>
        </Box>
      );
    }

    return (
      <TextField
        key={e.field}
        fullWidth
        label={e.field}
        value={typeof value === 'string' ? value : ''}
        onChange={(ev) => set(e.field, ev.target.value)}
        disabled={busy}
        error={invalidNumeric}
        helperText={
          invalidNumeric
            ? 'Must be a whole number'
            : e.description
        }
        sx={{ mb: 2 }}
      />
    );
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 12 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ maxWidth: 720 }}>
      <Typography variant="h5" component="h1" sx={{ fontWeight: 700, mb: 2 }}>
        Settings
      </Typography>

      {error && (
        <Alert severity="warning" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}
      {result === 'success' && (
        <Alert severity="success" sx={{ mb: 2 }}>
          {resultMessage}
        </Alert>
      )}
      {result === 'error' && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setResult(null)}>
          {resultMessage}
        </Alert>
      )}

      <Card variant="outlined">
        <CardContent>
          {sections.map((s, i) => (
            <Box key={s.section}>
              {i > 0 && <Divider sx={{ my: 2 }} />}
              <Typography variant="h6" component="h2" sx={{ fontWeight: 600, textTransform: 'capitalize', mb: 1.5 }}>
                {s.section}
              </Typography>
              {s.items.map(renderField)}
            </Box>
          ))}

          <Box sx={{ mt: 2, display: 'flex', alignItems: 'center', gap: 2 }}>
            <Button
              variant="contained"
              startIcon={busy ? <CircularProgress size={18} color="inherit" /> : null}
              disabled={!dirty || busy}
              onClick={() => void save()}
            >
              {busy ? 'Saving…' : 'Save settings'}
            </Button>
            {dirty && !busy && (
              <Typography variant="body2" color="text.secondary" sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                <Box component="span" sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: 'warning.main', display: 'inline-block' }} />
                Unsaved changes
              </Typography>
            )}
          </Box>
        </CardContent>
      </Card>
    </Box>
  );
}