import { app, BrowserWindow, ipcMain, Menu } from 'electron';
import * as path from 'path';
import { spawn } from 'child_process';

let mainWindow: BrowserWindow | null = null;

function createWindow(): void {
  Menu.setApplicationMenu(null);
  mainWindow = new BrowserWindow({
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
    mainWindow.loadURL('http://localhost:5173');
  } else {
    mainWindow.loadFile(path.join(__dirname, '..', 'dist', 'index.html'));
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

ipcMain.handle('otter:run', async (_event, command: string, args: string[], input?: string) => {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args);
    let stdout = '';
    let stderr = '';
    child.stdout.on('data', (d) => {
      stdout += d;
    });
    child.stderr.on('data', (d) => {
      stderr += d;
    });
    child.on('error', (err) => reject(err));
    child.on('close', (code, signal) => {
      const cleanErr = stripAnsi(stderr);
      if (code !== 0 || signal) {
        reject(new Error(cleanErr.trim() || `command exited with code ${String(code)}`));
        return;
      }
      resolve({ stdout: stripAnsi(stdout), stderr: cleanErr });
    });
    child.stdin.on('error', () => {});
    if (input !== undefined) {
      child.stdin.write(input);
    }
    child.stdin.end();
  });
});