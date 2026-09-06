import { useCallback, useEffect, useState } from 'react';
import Autocomplete from '@mui/material/Autocomplete';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import IconButton from '@mui/material/IconButton';
import Accordion from '@mui/material/Accordion';
import AccordionSummary from '@mui/material/AccordionSummary';
import AccordionDetails from '@mui/material/AccordionDetails';
import Alert from '@mui/material/Alert';
import Divider from '@mui/material/Divider';
import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import Switch from '@mui/material/Switch';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import AddCircleOutlinedIcon from '@mui/icons-material/AddCircleOutlined';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import type { RegistryEntry } from '../types';
import { prettyName } from '../distros';

const UNSHARE_FLAGS = [
  { key: 'unshareIpc', flag: '--unshare-ipc', label: 'IPC namespace' },
  { key: 'unshareNetns', flag: '--unshare-netns', label: 'Network namespace' },
  { key: 'unshareGroups', flag: '--unshare-groups', label: 'Additional groups' },
  { key: 'unshareDevsys', flag: '--unshare-devsys', label: 'Devices & sysfs' },
  { key: 'unshareProcess', flag: '--unshare-process', label: 'Process namespace' },
] as const;

const isWholeNumber = (value: string): boolean => /^[1-9]\d*$/.test(value.trim());

const isWholeMemory = (value: string): boolean => /^\d+[bkmgt]$/i.test(value.trim());

// Mirrors otter's makeContainerName when no name is given:
// a bare alias becomes "my-<alias>", an explicit ref is derived from its
// last path segment with ":" collapsed to "-".
function derivedName(image: string): string {
  if (image.includes('/') || image.includes(':')) {
    const base = image.split('/').pop() ?? image;
    return base.replaceAll(':', '-');
  }
  return `my-${image.toLowerCase()}`;
}

function imageBase(ref: string): string {
  return ref.split('/').pop() ?? ref;
}

// Expand a leading "~" or "~/" into the home directory. Electron spawns
// without a shell, so unlike the CLI (where the shell expands "~"), a literal
// "~" would reach podman as-is and be misread as a named volume name.
function expandHome(value: string, home: string): string {
  if (value === '~') return home;
  if (value.startsWith('~/')) return `${home}/${value.slice(2)}`;
  return value;
}

export default function Create() {
  const [images, setImages] = useState<RegistryEntry[]>([]);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [homeDir, setHomeDir] = useState('');

  const [name, setName] = useState('');
  const [image, setImage] = useState('');
  const [shell, setShell] = useState('bash');
  const [hostname, setHostname] = useState('');
  const [home, setHome] = useState('');
  const [platform, setPlatform] = useState('');
  const [memory, setMemory] = useState('');
  const [cpuThreads, setCpuThreads] = useState('');
  const [gpu, setGpu] = useState('mesa');
  const [volumes, setVolumes] = useState<string[]>([]);
  const [init, setInit] = useState(false);
  const [alwaysPull, setAlwaysPull] = useState(false);
  const [noEntry, setNoEntry] = useState(false);
  const [noUsernsLimit, setNoUsernsLimit] = useState(false);
  const [unshareAll, setUnshareAll] = useState(false);
  const [unshare, setUnshare] = useState<Record<string, boolean>>(
    Object.fromEntries(UNSHARE_FLAGS.map((f) => [f.key, false])),
  );
  const [initHooks, setInitHooks] = useState('');
  const [preInitHooks, setPreInitHooks] = useState('');
  const [additionalFlags, setAdditionalFlags] = useState('');

  const [busy, setBusy] = useState(false);
  const [step, setStep] = useState<string | null>(null);
  const [result, setResult] = useState<'success' | 'error' | null>(null);
  const [resultMessage, setResultMessage] = useState('');

  const load = useCallback(async () => {
    setLoadError(null);
    try {
      const { stdout, stderr } = await window.otter.run('otter', ['reg', 'list', '--json']);
      if (stderr) setLoadError(stderr);
      setImages(JSON.parse(stdout) as RegistryEntry[]);
    } catch (err) {
      const e = err as { message?: string; stderr?: string };
      setLoadError(e.stderr || e.message || 'Failed to load images');
    }
  }, []);

  useEffect(() => {
    void load();
    window.otter.homedir().then(setHomeDir).catch(() => {});
  }, [load]);

  const buildArgs = (json = false): string[] => {
    const args = ['create'];
    if (json) args.push('--json');
    if (name.trim()) args.push(name.trim());
    args.push('--image', image.trim());
    args.push('--shell', shell.trim());
    if (home.trim()) args.push('--home', expandHome(home.trim(), homeDir));
    if (platform.trim()) args.push('--platform', platform.trim());
    if (memory.trim()) args.push('--memory', memory.trim());
    if (cpuThreads.trim()) args.push('--cpu-threads', cpuThreads.trim());
    args.push('--gpu', gpu);
    for (const volume of volumes) {
      if (volume.trim()) {
        const [src] = volume.split(':');
        const expanded = src.trim().startsWith('~') ? `${expandHome(src.trim(), homeDir)}${volume.slice(src.length)}` : volume.trim();
        args.push('--volume', expanded);
      }
    }
    if (init) args.push('--init');
    if (alwaysPull) args.push('--always-pull');
    if (noEntry) args.push('--no-entry');
    if (noUsernsLimit) args.push('--no-userns-limit');
    if (unshareAll) {
      args.push('--unshare-all');
    } else {
      for (const f of UNSHARE_FLAGS) {
        if (unshare[f.key]) args.push(f.flag);
      }
    }
    if (initHooks.trim()) args.push('--init-hooks', initHooks.trim());
    if (preInitHooks.trim()) args.push('--pre-init-hooks', preInitHooks.trim());
    if (additionalFlags.trim()) args.push('--additional-flags', additionalFlags.trim());
    return args;
  };

  const submit = async () => {
    setResult(null);
    setResultMessage('');
    setBusy(true);
    setStep(null);
    const unsubscribe = window.otter.onRunStep((message) => {
      if (message !== null) setStep(message);
    });
    try {
      const { stderr } = await window.otter.runStream('otter', buildArgs(true));
      if (stderr) {
        setResult('error');
        setResultMessage(stderr);
      } else {
        setResult('success');
        const displayName = name.trim() || derivedName(image.trim());
        setResultMessage(`Container "${displayName}" created.`);
      }
    } catch (err) {
      const e = err as { message?: string; stderr?: string };
      setResult('error');
      setResultMessage(e.stderr || e.message || 'Failed to create container');
    } finally {
      unsubscribe();
      setBusy(false);
    }
  };

  const canSubmit = image.trim().length > 0 && !busy;
  const addVolume = () => setVolumes((v) => [...v, '']);
  const updateVolume = (i: number, value: string) =>
    setVolumes((v) => v.map((vol, idx) => (idx === i ? value : vol)));
  const removeVolume = (i: number) => setVolumes((v) => v.filter((_, idx) => idx !== i));

  return (
    <Box sx={{ maxWidth: 860 }}>
      <Typography variant="h5" component="h1" sx={{ fontWeight: 700 }}>
        Create container
      </Typography>
      <Typography color="text.secondary" variant="body2" sx={{ mb: 2 }}>
        Spin up a new otter container from a registry image
      </Typography>

      {loadError && (
        <Alert severity="warning" sx={{ mb: 2 }} onClose={() => setLoadError(null)}>
          {loadError}
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

      {busy ? (
        <Card variant="outlined" sx={{ py: 8, px: 3, textAlign: 'center' }}>
          <CircularProgress size={48} />
          <Typography variant="h6" sx={{ mt: 2, fontWeight: 600 }}>
            Creating container…
          </Typography>
          {step ? (
            <Typography
              variant="body2"
              color="text.secondary"
              sx={{ mt: 0.5, fontFamily: 'monospace', maxWidth: '100%', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
            >
              {step}
            </Typography>
          ) : (
            <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
              Pulling the image and setting up the container. This can take a while for large images.
            </Typography>
          )}
        </Card>
      ) : (
        <Card variant="outlined">
          <CardContent>
            <Box
              sx={{
                display: 'grid',
                gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' },
                gap: 2,
              }}
            >
              <TextField
                label="Container name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. my-alpinebox"
                helperText="Optional — a name is derived from the image if empty."
              />
              <Autocomplete
                freeSolo
                options={images.map((entry) => entry.name)}
                getOptionLabel={(opt) => opt}
                filterOptions={(opts, state) => {
                  const needle = state.inputValue.trim().toLowerCase();
                  if (!needle) return opts;
                  return opts.filter(
                    (opt) =>
                      opt.toLowerCase().includes(needle) ||
                      imageBase(images.find((i) => i.name === opt)?.image ?? '').includes(needle),
                  );
                }}
                renderOption={(props, opt) => {
                  const entry = images.find((e) => e.name === opt);
                  return (
                    <Box component="li" {...props} key={opt}>
                      {entry ? (
                        <span>
                          {prettyName(entry.name)}
                          <Box
                            component="span"
                            sx={{ fontFamily: 'monospace', ml: 1, color: 'text.secondary' }}
                          >
                            {entry.image}
                          </Box>
                        </span>
                      ) : (
                        <span>{opt}</span>
                      )}
                    </Box>
                  );
                }}
                renderInput={(params) => (
                  <TextField {...params} label="Image" required placeholder="alpine, ubuntu, fedora…" />
                )}
                value={image}
                onChange={(_e, value) => setImage(value ?? '')}
              />
              <FormControl fullWidth>
                <InputLabel>Shell</InputLabel>
                <Select
                  value={shell}
                  onChange={(e) => setShell(e.target.value)}
                  label="Shell"
                >
                  <MenuItem value="nu">nu</MenuItem>
                  <MenuItem value="bash">bash</MenuItem>
                  <MenuItem value="fish">fish</MenuItem>
                  <MenuItem value="zsh">zsh</MenuItem>
                </Select>
              </FormControl>
              <FormControl fullWidth>
                <InputLabel>GPU mode</InputLabel>
                <Select
                  value={gpu}
                  onChange={(e) => setGpu(e.target.value)}
                  label="GPU mode"
                >
                  <MenuItem value="mesa">Mesa</MenuItem>
                  <MenuItem value="nvidia">NVIDIA</MenuItem>
                  <MenuItem value="nvidia-toolkit">NVIDIA (toolkit)</MenuItem>
                </Select>
              </FormControl>
              <TextField
                label="Home directory"
                value={home}
                onChange={(e) => setHome(e.target.value)}
                placeholder="e.g. /home/user/otter-box"
              />
              <FormControl fullWidth>
                <InputLabel>Platform</InputLabel>
                <Select
                  value={platform}
                  onChange={(e) => setPlatform(e.target.value)}
                  label="Platform"
                >
                  <MenuItem value="linux/amd64">linux/amd64</MenuItem>
                  <MenuItem value="linux/arm64">linux/arm64</MenuItem>
                </Select>
              </FormControl>
            </Box>

            <Stack direction="row" spacing={3} sx={{ mt: 2, flexWrap: 'wrap' }}>
              <FormControlLabel
                control={<Switch checked={init} onChange={(e) => setInit(e.target.checked)} />}
                label="Use init system"
              />
              <FormControlLabel
                control={<Switch checked={alwaysPull} onChange={(e) => setAlwaysPull(e.target.checked)} />}
                label="Always pull the image"
              />
            </Stack>

            <Accordion
              elevation={0}
              sx={{
                mt: 2,
                border: 1,
                borderColor: 'divider',
                borderRadius: 2,
                '&::before': { display: 'none' },
              }}
            >
              <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                Advanced options
              </AccordionSummary>
              <AccordionDetails>
                <Box
                  sx={{
                    display: 'grid',
                    gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' },
                    gap: 2,
                    mb: 2,
                  }}
                >
                  <TextField
                    label="Memory limit"
                    value={memory}
                    onChange={(e) => setMemory(e.target.value)}
                    placeholder="e.g. 2g or 2048m"
                    error={memory.length > 0 && !isWholeMemory(memory)}
                    helperText={
                      memory.length > 0 && !isWholeMemory(memory)
                        ? 'Whole number with a unit (e.g. 512m, 2g)'
                        : ' '
                    }
                  />
                  <TextField
                    label="CPU threads"
                    value={cpuThreads}
                    onChange={(e) => setCpuThreads(e.target.value)}
                    placeholder="e.g. 4"
                    inputMode="numeric"
                    error={cpuThreads.length > 0 && !isWholeNumber(cpuThreads)}
                    helperText={
                      cpuThreads.length > 0 && !isWholeNumber(cpuThreads)
                        ? 'Must be a whole number'
                        : ' '
                    }
                  />
                </Box>

                <TextField
                  label="Hostname"
                  fullWidth
                  value={hostname}
                  onChange={(e) => setHostname(e.target.value)}
                  sx={{ mb: 2 }}
                />

                <Typography variant="subtitle2" sx={{ mt: 1, mb: 1 }}>
                  Volumes
                </Typography>
                {volumes.map((volume, i) => (
                  <Box key={i} sx={{ display: 'flex', gap: 1, mb: 1, alignItems: 'center' }}>
                    <TextField
                      fullWidth
                      placeholder="/host/path:/container/path:rw"
                      value={volume}
                      onChange={(e) => updateVolume(i, e.target.value)}
                    />
                    <IconButton aria-label="Remove volume" onClick={() => removeVolume(i)}>
                      <DeleteOutlinedIcon />
                    </IconButton>
                  </Box>
                ))}
                <Button size="small" startIcon={<AddCircleOutlinedIcon />} onClick={addVolume}>
                  Add volume
                </Button>

                <Typography variant="subtitle2" sx={{ mt: 3, mb: 1 }}>
                  Isolation
                </Typography>
                <FormControlLabel
                  control={<Switch checked={unshareAll} onChange={(e) => setUnshareAll(e.target.checked)} />}
                  label="Unshare all namespaces"
                />
                {UNSHARE_FLAGS.map((f) => (
                  <FormControlLabel
                    key={f.key}
                    control={
                      <Switch
                        checked={unshare[f.key]}
                        onChange={(e) =>
                          setUnshare((u) => ({ ...u, [f.key]: e.target.checked }))
                        }
                        disabled={unshareAll}
                      />
                    }
                    label={`Do not share ${f.label}`}
                    sx={{ display: 'flex' }}
                  />
                ))}

                <Typography variant="subtitle2" sx={{ mt: 3, mb: 1 }}>
                  Hooks
                </Typography>
                <TextField
                  label="Init hooks"
                  fullWidth
                  value={initHooks}
                  onChange={(e) => setInitHooks(e.target.value)}
                  placeholder="commands run at the end of initialization"
                  sx={{ mb: 1.5 }}
                />
                <TextField
                  label="Pre-init hooks"
                  fullWidth
                  value={preInitHooks}
                  onChange={(e) => setPreInitHooks(e.target.value)}
                  placeholder="commands run at the start of initialization"
                />

                <Divider sx={{ my: 2 }} />

                <FormControlLabel
                  control={<Switch checked={noEntry} onChange={(e) => setNoEntry(e.target.checked)} />}
                  label="No desktop entry"
                />
                <FormControlLabel
                  control={<Switch checked={noUsernsLimit} onChange={(e) => setNoUsernsLimit(e.target.checked)} />}
                  label="No userns limit (podman)"
                />
                <TextField
                  label="Additional flags"
                  fullWidth
                  value={additionalFlags}
                  onChange={(e) => setAdditionalFlags(e.target.value)}
                  placeholder="extra flags passed to the container manager"
                  sx={{ mt: 2 }}
                />
              </AccordionDetails>
            </Accordion>

            <Box sx={{ mt: 3, display: 'flex', alignItems: 'center', gap: 2 }}>
              <Button
                variant="contained"
                startIcon={<AddCircleOutlinedIcon />}
                disabled={!canSubmit}
                onClick={() => void submit()}
              >
                Create container
              </Button>
            </Box>
          </CardContent>
        </Card>
      )}
    </Box>
  );
}