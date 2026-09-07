import { app, BrowserWindow, ipcMain, Menu } from 'electron';
import * as os from 'os';
import * as path from 'path';
import { spawn } from 'child_process';

function createWindow(): void {
  Menu.setApplicationMenu(null);
  const win = new BrowserWindow({
    width: 1200,
    height: 800,
    minWidth: 800,
    minHeight: 600,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
    },
  });

  const dev = process.argv.includes('--dev');
  if (dev) {
    win.loadURL('http://localhost:5173');
  } else {
    win.loadFile(path.join(__dirname, '..', 'dist', 'index.html'));
  }
}

app.whenReady().then(() => {
  createWindow();
  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit();
});

const stripAnsi = (s: string): string => s.replace(/\u001b\[[0-9;]*m/g, '');

interface LogEntry {
  id: number;
  ts: number;
  command: string;
  args: string[];
  input?: string;
  status: 'running' | 'ok' | 'error';
  exitCode?: number;
  durationMs?: number;
  stdout?: string;
  stderr?: string;
}

const LOG_CAP = 300;
let logId = 0;
const otterLog: LogEntry[] = [];

const broadcastLog = (entry: LogEntry): void => {
  for (const win of BrowserWindow.getAllWindows()) win.webContents.send('otter:log', entry);
};

const appendLog = (entry: LogEntry): void => {
  otterLog.push(entry);
  while (otterLog.length > LOG_CAP) otterLog.shift();
  broadcastLog(entry);
};

// Replicate the shell's argument expansion (which a terminal applies before
// otter ever runs): $VAR / ${VAR} from the main process env, plus a leading
// "~" into the user's home directory. holt spawns without a shell, so this
// restores the behaviour CLI users get for free.
const expandEnv = (value: string): string =>
  value.replace(/\$\{(\w+)\}|\$(\w+)/g, (match, braced: string, plain: string) => {
    const name = braced || plain;
    return process.env[name] !== undefined ? process.env[name] : match;
  });

const expandValue = (value: string): string => {
  const expanded = expandEnv(value);
  if (expanded === '~') return os.homedir();
  if (expanded.startsWith('~/')) return `${os.homedir()}${expanded.slice(1)}`;
  return expanded;
};

ipcMain.handle('otter:expand-env', (_event, value: string) => expandValue(value));

ipcMain.handle('otter:log:get', () => otterLog.slice());
ipcMain.handle('otter:log:clear', () => {
  otterLog.length = 0;
});

interface RunOptions {
  event?: Electron.IpcMainInvokeEvent;
  command: string;
  args: string[];
  input?: string;
  onStdoutLine?: (line: string) => void;
}

const runCommand = (options: RunOptions): Promise<{ stdout: string; stderr: string }> => {
  const { event, command, args, input, onStdoutLine } = options;
  const win = event ? BrowserWindow.fromWebContents(event.sender) : undefined;
  return new Promise((resolve, reject) => {
    const entry: LogEntry = {
      id: logId++,
      ts: Date.now(),
      command,
      args,
      ...(input === undefined ? {} : { input }),
      status: 'running',
    };
    broadcastLog(entry);

    const sendStep = (message: string | null): void => {
      if (onStdoutLine) win?.webContents.send('otter:run-step', message);
    };

    let settled = false;
    const finish = (status: 'ok' | 'error', exitCode: number | undefined, stdout: string, stderr: string) => {
      entry.status = status;
      entry.exitCode = exitCode;
      entry.durationMs = Date.now() - entry.ts;
      entry.stdout = stripAnsi(stdout);
      entry.stderr = stripAnsi(stderr);
      appendLog(entry);
    };

    const child = spawn(command, args);
    let stdout = '';
    let stderr = '';
    let lineBuffer = '';

    child.stdout.on('data', (d) => {
      const chunk = d.toString();
      stdout += chunk;
      if (!onStdoutLine) return;
      lineBuffer += chunk;
      const lines = lineBuffer.split('\n');
      lineBuffer = lines.pop() ?? '';
      for (const line of lines) {
        const trimmed = line.trim();
        if (trimmed) onStdoutLine(trimmed);
      }
    });

    child.stderr.on('data', (d) => {
      stderr += d.toString();
    });

    child.on('error', (err) => {
      if (settled) return;
      settled = true;
      sendStep(null);
      finish('error', undefined, '', err.message);
      reject(new Error(err.message));
    });

    child.on('close', (code, signal) => {
      if (settled) return;
      settled = true;
      if (onStdoutLine) {
        if (lineBuffer.trim()) onStdoutLine(lineBuffer.trim());
        sendStep(null);
      }
      if (code !== 0 || signal) {
        finish('error', code ?? undefined, stdout, stderr);
        reject(new Error(entry.stderr?.trim() || `command exited with code ${String(code)}`));
        return;
      }
      finish('ok', 0, stdout, stderr);
      resolve({ stdout: entry.stdout ?? '', stderr: entry.stderr ?? '' });
    });

    child.stdin.on('error', () => {});
    if (input !== undefined) {
      child.stdin.write(input);
    }
    child.stdin.end();
  });
};

ipcMain.handle('otter:run', (_event, command: string, args: string[], input?: string) =>
  runCommand({ command, args, input }),
);

ipcMain.handle('otter:run-stream', (event, command: string, args: string[], input?: string) =>
  runCommand({
    event,
    command,
    args,
    input,
    onStdoutLine: (line) => {
      try {
        const obj = JSON.parse(line) as Record<string, unknown>;
        if (typeof obj.message === 'string') {
          BrowserWindow.fromWebContents(event.sender)?.webContents.send('otter:run-step', obj.message);
        }
      } catch {
        // not JSON — ignore
      }
    },
  }),
);