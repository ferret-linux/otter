import { contextBridge, ipcRenderer } from 'electron';

const api = {
  run: (command: string, args: string[], input?: string) =>
    ipcRenderer.invoke('otter:run', command, args, input),
  getLog: () => ipcRenderer.invoke('otter:log:get'),
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