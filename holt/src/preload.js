const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('otter', {
  run: (command, args) => ipcRenderer.invoke('otter:run', command, args),
});
