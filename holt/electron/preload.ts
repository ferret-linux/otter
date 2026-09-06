import { contextBridge, ipcRenderer } from 'electron';

const api = {
  run: (command: string, args: string[], input?: string) =>
    ipcRenderer.invoke('otter:run', command, args, input),
};

contextBridge.exposeInMainWorld('otter', api);