import { app, BrowserWindow, ipcMain, Menu } from 'electron';
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

ipcMain.handle('otter:log:get', () => otterLog.slice());
ipcMain.handle('otter:log:clear', () => {
  otterLog.length = 0;
});

ipcMain.handle('otter:run', async (_event, command: string, args: string[], input?: string) => {
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
    child.stdout.on('data', (d) => {
      stdout += d;
    });
    child.stderr.on('data', (d) => {
      stderr += d;
    });
    child.on('error', (err) => {
      if (settled) return;
      settled = true;
      finish('error', undefined, '', err.message);
      reject(new Error(err.message));
    });
    child.on('close', (code, signal) => {
      if (settled) return;
      settled = true;
      const cleanErr = stripAnsi(stderr);
      if (code !== 0 || signal) {
        finish('error', code ?? undefined, stdout, stderr);
        reject(new Error(cleanErr.trim() || `command exited with code ${String(code)}`));
        return;
      }
      finish('ok', 0, stdout, stderr);
      resolve({ stdout: stripAnsi(stdout), stderr: cleanErr });
    });
    child.stdin.on('error', () => {});
    if (input !== undefined) {
      child.stdin.write(input);
    }
    child.stdin.end();
  });
});