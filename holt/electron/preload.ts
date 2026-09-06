import { contextBridge, ipcRenderer } from 'electron';

const api = {
  run: (command: string, args: string[]) => ipcRenderer.invoke('otter:run', command, args),
};

contextBridge.exposeInMainWorld('otter', api);