import { app, BrowserWindow, ipcMain } from 'electron';
import * as path from 'path';
import { execFile } from 'child_process';

let mainWindow: BrowserWindow | null = null;

function createWindow(): void {
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

ipcMain.handle('otter:run', async (_event, command: string, args: string[]) => {
  return new Promise((resolve, reject) => {
    execFile(command, args, { timeout: 30_000 }, (error, stdout, stderr) => {
      if (error) {
        reject({ message: error.message, stderr });
      } else {
        resolve({ stdout, stderr });
      }
    });
  });
});