import { contextBridge, ipcRenderer } from 'electron';

const api = {
  run: (command: string, args: string[], input?: string) =>
    ipcRenderer.invoke('otter:run', command, args, input),
  runStream: (command: string, args: string[], input?: string) =>
    ipcRenderer.invoke('otter:run-stream', command, args, input),
  onRunStep: (callback: (message: string | null) => void) => {
    const listener = (_event: unknown, message: string | null) => callback(message);
    ipcRenderer.on('otter:run-step', listener);
    return () => {
      ipcRenderer.removeListener('otter:run-step', listener);
    };
  },
  getLog: () => ipcRenderer.invoke('otter:log:get'),
  expandEnv: (value: string) => ipcRenderer.invoke('otter:expand-env', value),
  clearLog: () => ipcRenderer.invoke('otter:log:clear'),
  onLog: (callback: (entry: unknown) => void) => {
    const listener = (_event: unknown, entry: unknown) => callback(entry);
    ipcRenderer.on('otter:log', listener);
    return () => {
      ipcRenderer.removeListener('otter:log', listener);
    };
  },
};

contextBridge.exposeInMainWorld('otter', api);