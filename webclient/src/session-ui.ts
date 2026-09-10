import Connection from './connection';
import {configureSession, sessionConfig} from './session';
import {ready} from './globals';
import FileTransferClient from './file-transfer';
import {FileType} from './message';
const status = document.querySelector('#status') as HTMLElement;
const screen = document.querySelector('#screen') as HTMLCanvasElement;
const passwordForm = document.querySelector('#password-form') as HTMLFormElement;
const password = document.querySelector('#password') as HTMLInputElement;
const display = document.querySelector('#display') as HTMLSelectElement;
let conn: Connection;
let active = false;
let displays: any[] = [];
let displayIndex = 0;
let viewOnly = false;
let blocked = false;
let fileConn: Connection | undefined;
let transfer: FileTransferClient | undefined;
const $ = <T extends HTMLElement = HTMLElement>(id: string) => document.getElementById(id) as T;
const button = (id: string) => $<HTMLButtonElement>(id);
const input = (id: string) => $<HTMLInputElement>(id);
const notice = (text: string) => { $('control-status').textContent = text; };
function controls() {
  const connected = !!conn?.authenticated;
  const keyboard = connected && conn.permissions.keyboard && !viewOnly;
  for (const id of ['ctrl-alt-del','lock-screen']) button(id).disabled = !keyboard;
  for (const id of ['refresh','quality','display','remote-cursor','lock-after']) ($<HTMLInputElement>(id)).disabled = !connected;
  button('block-input').disabled = !keyboard || conn?._peerInfo?.platform !== 'Windows' || !conn?.permissions.blockInput;
  input('lock-after').disabled = !keyboard || conn?._peerInfo?.platform === 'Android';
  button('clipboard').disabled = !connected;
  button('send-clipboard').disabled = !connected || !conn.permissions.clipboard;
  button('type-text').disabled = !keyboard;
  const platform = conn?._peerInfo?.platform || '';
  if (conn?._peerInfo?.platform !== 'Linux' && !(platform === 'Windows' && conn?._peerInfo?.sas_enabled)) button('ctrl-alt-del').disabled = true;
  for (const el of document.querySelectorAll<HTMLElement>('[data-host-action]')) { el.hidden = platform !== 'Android'; (el as HTMLButtonElement).disabled = !keyboard; }
  $('android-controls').hidden = platform !== 'Android'; $('android-navigation').hidden = platform !== 'Android'; $('peer-platform').textContent = conn?._peerInfo ? `Remote ${platform} · RustDesk ${conn._peerInfo.version || 'unknown'}` : 'Waiting for host';
}
const player = (window as any).YUVCanvas.attach(screen);
function modifiers(e: MouseEvent | KeyboardEvent) { return [e.altKey,e.ctrlKey,e.shiftKey,e.metaKey] as const; }
function stop(text: string) { active = false; conn?.close(); fileConn?.close(); transfer?.close(); status.textContent = text; passwordForm.hidden = true; controls(); }
window.addEventListener('rustdesk-status', ((e: CustomEvent) => {
  const {type,title,text} = e.detail;
  status.textContent = text || title || 'Connected';
  passwordForm.hidden = !['input-password','re-input-password'].includes(type);
  if (!passwordForm.hidden) password.focus();
  if (type === 'error') stop(text || title);
}) as EventListener);
window.addEventListener('rustdesk-frame', ((e: CustomEvent) => {
  player.drawFrame(e.detail);
  screen.hidden = false;
  if (!active) { active = true; screen.focus(); }
  status.textContent = 'Connected — click the desktop to control';
}) as EventListener);
window.addEventListener('rustdesk-peer_info', ((e: CustomEvent) => {
  displays = e.detail.displays;
  displayIndex = e.detail.current_display || 0;
  display.replaceChildren(...displays.map((d, i) => {
    const opt = document.createElement('option'); opt.value = String(i); opt.textContent = `Display ${i + 1} (${d.width} × ${d.height})`; return opt;
  }));
  display.value = String(displayIndex); controls();
}) as EventListener);
window.addEventListener('rustdesk-switch_display', ((e: CustomEvent) => { displayIndex = e.detail.display || 0; display.value = String(displayIndex); }) as EventListener);
window.addEventListener('rustdesk-permission', (() => { controls(); if (!conn.permissions.keyboard) releaseInputs(); }) as EventListener);
window.addEventListener('rustdesk-closed', (() => { active = false; controls(); }) as EventListener);
passwordForm.addEventListener('submit', e => {
  e.preventDefault(); const value = password.value; password.value = ''; passwordForm.hidden = true; conn.login(value);
});
display.addEventListener('change', () => { displayIndex = Number(display.value); conn.switchDisplay(displayIndex); screen.focus(); });
document.querySelector('#disconnect')!.addEventListener('click', async () => {
  stop('Disconnecting…');
  try {
    const response = await fetch('/webclient/logout', {method:'POST',headers:{'Content-Type':'application/json'},body:'{}',credentials:'same-origin'});
    if (!response.ok) throw new Error('Could not revoke the access session. Close this page and try again.');
    status.textContent = 'Disconnected. Open a new access link to reconnect.';
  } catch (e: any) { status.textContent = e.message; }
});
document.querySelector('#fullscreen')!.addEventListener('click', async () => {
  try { if (document.fullscreenElement) await document.exitFullscreen(); else await document.body.requestFullscreen(); } catch { status.textContent = 'Full screen is unavailable in this browser.'; }
});
button('ctrl-alt-del').addEventListener('click', () => { conn?.ctrlAltDel(); screen.focus(); });
document.querySelectorAll<HTMLButtonElement>('[data-host-action]').forEach(el => el.addEventListener('click', () => { conn?.mobileAction(el.dataset.hostAction as any); screen.focus(); }));
button('show-controls').addEventListener('click', () => $<HTMLDialogElement>('controls-dialog').showModal());
button('lock-screen').addEventListener('click', () => conn?.lockScreen());
button('block-input').addEventListener('click', () => {
  button('block-input').disabled = true;
  conn?.toggleOption(blocked ? 'unblock-input' : 'block-input');
  notice('Waiting for remote input control confirmation…');
  setTimeout(() => { if (button('block-input').disabled && conn?.authenticated && !viewOnly) { controls(); notice('No input-control confirmation received from the host.'); } }, 5000);
});
window.addEventListener('rustdesk-back_notification', ((e: CustomEvent) => {
  const state = e.detail.block_input_state;
  if (state === undefined) return;
  if (state === 2) blocked = true;
  if (state === 4) blocked = false;
  button('block-input').textContent = blocked ? 'Unblock remote input' : 'Block remote input';
  notice(state === 2 || state === 4 ? 'Remote input setting updated.' : 'The remote host could not change input blocking.');
  controls();
}) as EventListener);
button('refresh').addEventListener('click', () => conn?.refresh());
$('quality').addEventListener('change', () => conn?.setImageQuality(input('quality').value));
$('view').addEventListener('change', () => $('viewport').classList.toggle('original', input('view').value === 'original'));
input('view-only').addEventListener('change', () => { releaseInputs(); viewOnly = input('view-only').checked; controls(); });
input('remote-cursor').addEventListener('change', () => conn?.toggleOption('show-remote-cursor'));
input('lock-after').addEventListener('change', () => conn?.toggleOption('lock-after-session-end'));
for (const close of document.querySelectorAll<HTMLButtonElement>('[data-close]')) close.addEventListener('click', () => $<HTMLDialogElement>(close.dataset.close!).close());
button('clipboard').addEventListener('click', () => $<HTMLDialogElement>('clipboard-dialog').showModal());
window.addEventListener('rustdesk-clipboard', ((e: CustomEvent) => {
  $<HTMLTextAreaElement>('clipboard-text').value = e.detail.text;
  $('clipboard-status').textContent = 'Remote clipboard received. Use “Copy received text locally” to copy it.';
}) as EventListener);
async function clipboardAction(action: () => void | Promise<void>) {
  try { await action(); $('clipboard-status').textContent = 'Done.'; } catch (error: any) { $('clipboard-status').textContent = error.message || 'Clipboard permission denied.'; }
}
button('read-local').addEventListener('click', () => clipboardAction(async () => { $<HTMLTextAreaElement>('clipboard-text').value = await navigator.clipboard.readText(); }));
button('send-clipboard').addEventListener('click', () => clipboardAction(() => conn.sendClipboard(input('clipboard-text').value)));
button('type-text').addEventListener('click', () => { if (!viewOnly) clipboardAction(() => conn.inputString(input('clipboard-text').value)); });
button('copy-remote').addEventListener('click', () => clipboardAction(() => navigator.clipboard.writeText(input('clipboard-text').value)));
const pressedKeys = new Map<string,string>();
const controlKeys: Record<string,string> = {Enter:'Return',Escape:'Escape',Backspace:'Backspace',Tab:'Tab',Delete:'Delete',Insert:'Insert',Home:'Home',End:'End',PageUp:'PageUp',PageDown:'PageDown',ArrowUp:'UpArrow',ArrowDown:'DownArrow',ArrowLeft:'LeftArrow',ArrowRight:'RightArrow',Shift:'Shift',Control:'Control',Alt:'Alt',Meta:'Meta',CapsLock:'CapsLock'};
function keyName(e: KeyboardEvent) { return controlKeys[e.key] || e.key; }
screen.addEventListener('keydown', e => {
  if (!active || viewOnly || !conn.permissions.keyboard) return; e.preventDefault(); const key = keyName(e);
  if (e.repeat && pressedKeys.has(e.code)) return;
  pressedKeys.set(e.code,key); conn.inputKey(key,true,false,...modifiers(e));
});
screen.addEventListener('keyup', e => {
  if (!active || viewOnly || !conn.permissions.keyboard) return; e.preventDefault(); conn.inputKey(pressedKeys.get(e.code) || keyName(e),false,false,...modifiers(e)); pressedKeys.delete(e.code);
});
screen.addEventListener('blur', releaseInputs);
const mouseButtons = new Set<number>();
function releaseInputs() {
  for (const key of pressedKeys.values()) conn?.inputKey(key,false,false,false,false,false,false);
  pressedKeys.clear();
  for (const b of mouseButtons) conn?.inputMouse(2 | b << 3);
  mouseButtons.clear();
}
function coordinates(e: PointerEvent) {
  const rect = screen.getBoundingClientRect(); const d = displays[displayIndex] || {x:0,y:0};
  return [Math.round((e.clientX-rect.left)*screen.width/rect.width)+(d.x||0),Math.round((e.clientY-rect.top)*screen.height/rect.height)+(d.y||0)] as const;
}
screen.addEventListener('pointermove', e => { if(active && !viewOnly && conn.permissions.keyboard) conn.inputMouse(0,...coordinates(e),...modifiers(e)); });
screen.addEventListener('pointerdown', e => {
  if (!active || viewOnly || !conn.permissions.keyboard) return; e.preventDefault(); screen.focus(); screen.setPointerCapture(e.pointerId);
  const button = [1,4,2][e.button] || 1; mouseButtons.add(button); conn.inputMouse(1 | button << 3,...coordinates(e),...modifiers(e));
});
screen.addEventListener('pointerup', e => {
  if (!active || viewOnly || !conn.permissions.keyboard) return; e.preventDefault(); const button = [1,4,2][e.button] || 1; mouseButtons.delete(button); conn.inputMouse(2 | button << 3,...coordinates(e),...modifiers(e));
});
screen.addEventListener('lostpointercapture', () => { for(const b of mouseButtons) conn?.inputMouse(2 | b << 3); mouseButtons.clear(); });
screen.addEventListener('contextmenu', e => e.preventDefault());
screen.addEventListener('wheel', e => { if(active && !viewOnly && conn.permissions.keyboard) { e.preventDefault(); conn.inputMouse(3,-Math.sign(e.deltaX),-Math.sign(e.deltaY),...modifiers(e)); } },{passive:false});
window.addEventListener('beforeunload', () => { conn?.close(); fileConn?.close(); transfer?.close(); });
type Entry = {name:string; directory:boolean; size:number; modified:number; path:string; file?:File; handle?:any};
type QueueItem = {name:string; path:string; file?:File; direction:'upload'|'download'; state:string; row:HTMLLIElement; status:HTMLElement; url?:string};
let folder = '';
let remoteEntries: Entry[] = [];
let localEntries: Entry[] = [];
let localFiles: File[] = [];
let localFolder = '';
let localHandles: any[] = [];
let localHistory: {folder:string; handles:any[]}[] = [];
let localLoadGeneration = 0;
let remoteHistory: string[] = [];
let remoteBack = false;
let remoteLoaded = false;
const selections = {local:new Set<string>(), remote:new Set<string>()};
const sorting = {local:{key:'name', descending:false},remote:{key:'name',descending:false}};
let queue: QueueItem[] = [];
let currentJob: QueueItem | undefined;
function bytes(size:number) { return size < 1024 ? `${size} B` : size < 1048576 ? `${(size/1024).toFixed(1)} KB` : `${(size/1048576).toFixed(1)} MB`; }
function fileControls() {
  const connected = !!fileConn?.authenticated && fileConn.permissions.file;
  const busy = !!transfer?.busy || !!currentJob;
  for (const id of ['browse-files','files-home','files-up','files-refresh','show-hidden']) input(id).disabled = !connected || busy;
  button('files-back').disabled = !connected || busy || !remoteHistory.length;
  button('send-files').disabled = !connected || !remoteLoaded || !selections.local.size;
  button('receive-files').disabled = !connected || !selections.remote.size;
  button('cancel-transfer').disabled = !busy;
  button('retry-files').disabled = busy;
  button('local-up').disabled = !localFolder && localHandles.length < 2;
  button('local-back').disabled = !localHistory.length;
  button('local-home').disabled = !localFolder && localHandles.length < 2;
  for (const action of $('file-list').querySelectorAll<HTMLButtonElement>('button')) action.disabled = !connected || busy;
}
function fileAction(action: () => void | Promise<void>) {
  Promise.resolve().then(action).catch(error => { $('files-status').textContent = error.message; }).finally(fileControls);
}
function remotePath(name: string) {
  const separator = fileConn?._peerInfo?.platform === 'Windows' ? '\\' : '/';
  return folder.replace(/[\\/]$/, '') + separator + name;
}
function renderEntries(side:'local'|'remote') {
  const selected = selections[side];
  const query = input(`${side}-search`).value.toLocaleLowerCase();
  const sort = sorting[side];
  const entries = (side === 'local' ? localEntries : remoteEntries).filter(e => e.name.toLocaleLowerCase().includes(query)).sort((a,b) => {
    if (a.directory !== b.directory) return a.directory ? -1 : 1;
    const cmp = sort.key === 'name' ? a.name.localeCompare(b.name,undefined,{numeric:true}) : sort.key === 'size' ? a.size - b.size : a.modified - b.modified;
    return sort.descending ? -cmp : cmp;
  });
  const rows = entries.map(entry => {
    const row = document.createElement('tr'); row.setAttribute('aria-selected', String(selected.has(entry.path)));
    const nameCell = document.createElement('td'); nameCell.title = entry.name;
    const label = document.createElement('div'); label.className = 'entry-name';
    const icon = document.createElement('span'); icon.textContent = entry.directory ? '▰' : '▤'; icon.setAttribute('aria-hidden','true');
    const name = document.createElement(entry.directory ? 'button' : 'span'); name.textContent = entry.name;
    const open = () => { if(side === 'remote' && (!fileConn?.authenticated || transfer?.busy || currentJob)) return; fileAction(() => side === 'local' ? openLocal(entry) : transfer!.list(entry.path,input('show-hidden').checked)); };
    if (entry.directory) { name.setAttribute('aria-label',`Open ${side} folder ${entry.name}`); name.addEventListener('click',open); row.addEventListener('dblclick',open); }
    else {
      const check = document.createElement('input'); check.type = 'checkbox'; check.checked = selected.has(entry.path); check.setAttribute('aria-label',`Select ${side} file ${entry.name}`);
      check.addEventListener('change',() => { if(check.checked) selected.add(entry.path); else selected.delete(entry.path); row.setAttribute('aria-selected',String(check.checked)); updateCount(side); fileControls(); }); label.append(check);
    }
    label.append(icon,name); nameCell.append(label);
    const modified = document.createElement('td'); modified.textContent = entry.modified ? new Date(entry.modified).toLocaleString() : '—'; modified.title = modified.textContent;
    const size = document.createElement('td'); size.textContent = entry.directory ? '' : bytes(entry.size);
    row.append(nameCell,modified,size); return row;
  });
  $(side === 'local' ? 'local-file-list' : 'file-list').replaceChildren(...rows);
  $(`${side}-empty`).hidden = rows.length > 0;
  $(`${side}-empty`).textContent = query ? 'No matching files.' : side === 'local' && !localFiles.length && !localHandles.length ? 'Choose files or a folder to browse this computer.' : 'This folder is empty.';
  updateCount(side); fileControls();
}
function updateCount(side:'local'|'remote') {
  const entries = side === 'local' ? localEntries : remoteEntries;
  $(`${side}-count`).textContent = `${entries.length} items · ${selections[side].size} selected`;
}
async function loadLocal() {
  const generation = ++localLoadGeneration;
  const handle = localHandles.at(-1);
  if (handle) {
    const entries:Entry[] = [];
    for await (const item of handle.values()) {
      const file = item.kind === 'file' ? await item.getFile() : undefined;
      entries.push({name:item.name,directory:!file,size:file?.size||0,modified:file?.lastModified||0,path:item.name,file,handle:item});
    }
    if(generation !== localLoadGeneration) return;
    localEntries = entries;
    input('local-path').value = localHandles.map(h => h.name).join('/');
  } else {
    const prefix = localFolder ? localFolder + '/' : '';
    const entries = new Map<string,Entry>();
    for (const file of localFiles) {
      const path = file.webkitRelativePath || file.name;
      if (!path.startsWith(prefix)) continue;
      const rest = path.slice(prefix.length); const name = rest.split('/')[0]; if (!name) continue;
      const directory = rest.includes('/');
      entries.set(name,{name,directory,size:directory?0:file.size,modified:directory?0:file.lastModified,path:prefix+name,file:directory?undefined:file});
    }
    localEntries = [...entries.values()]; input('local-path').value = localFolder || 'Selected files';
  }
  selections.local.clear(); renderEntries('local');
}
async function openLocal(entry:Entry) {
  localHistory.push({folder:localFolder,handles:[...localHandles]});
  if(entry.handle) localHandles.push(entry.handle); else localFolder = entry.path;
  await loadLocal();
}
function finishJob(state:string) {
  if (!currentJob) return;
  currentJob.state = state; currentJob.status.textContent = state; currentJob = undefined;
  queueMicrotask(runQueue);
}
function runQueue() {
  if (currentJob || transfer?.busy || !fileConn?.authenticated || !fileConn.permissions.file) return;
  const item = queue.find(j => j.state === 'Queued');
  if (!item) { fileControls(); return; }
  currentJob = item; item.state = 'Transferring'; item.status.textContent = item.state;
  try { if(item.file) transfer!.upload(item.path,item.file); else transfer!.download(item.path); }
  catch(error:any) { finishJob(error.message); }
  fileControls();
}
function enqueue(direction:'upload'|'download') {
  const side = direction === 'upload' ? 'local' : 'remote';
  const entries = (side === 'local' ? localEntries : remoteEntries).filter(e => !e.directory && selections[side].has(e.path));
  for (const entry of entries) {
    const row = document.createElement('li'); const name = document.createElement('strong'); name.textContent = `${direction === 'upload' ? '→' : '←'} ${entry.name}`;
    const info = document.createElement('small'); info.textContent = `${direction === 'upload' ? 'Send to remote' : 'Receive from remote'} · ${bytes(entry.size)}`;
    const status = document.createElement('span'); status.textContent = 'Queued';
    const item:QueueItem = {name:entry.name,path:direction === 'upload' ? folder : entry.path,file:direction === 'upload' ? entry.file : undefined,direction,state:'Queued',row,status};
    row.append(name,info,status); $('transfer-queue').append(row); queue.push(item);
  }
  selections[side].clear(); renderEntries(side); $('queue-empty').hidden = !!queue.length; runQueue();
}
function fileEvent(name: string, detail: any) {
  if (name === 'directory') {
    if(remoteLoaded && folder !== detail.path && !remoteBack) remoteHistory.push(folder);
    remoteBack = false; folder = detail.path; remoteLoaded = true;
    input('remote-path').value = folder;
    remoteEntries = detail.entries.map((entry:any) => ({name:entry.name,directory:[FileType.Dir,FileType.DirLink,FileType.DirDrive].includes(entry.entry_type),size:entry.size,modified:entry.modified_time*1000,path:entry.entry_type === FileType.DirDrive ? entry.name.replace(/[\\/]?$/, '\\') : remotePath(entry.name)})).filter((entry:Entry,index:number) => entry.directory || detail.entries[index].entry_type === FileType.File);
    selections.remote.clear(); renderEntries('remote');
  } else if (name === 'status') {
    $('files-status').textContent = detail.text;
    if (!detail.busy) $('file-progress').hidden = true;
    if (detail.error && currentJob && !detail.busy) finishJob(detail.text);
  } else if (name === 'progress') {
    const progress = $<HTMLProgressElement>('file-progress'); progress.hidden = false; progress.value = detail.total ? detail.transferred / detail.total : 1;
    if(currentJob) currentJob.status.textContent = `${bytes(detail.transferred)} / ${bytes(detail.total)}`;
  } else if (name === 'download' && currentJob) {
    const link = document.createElement('a'); currentJob.url = URL.createObjectURL(detail.blob); link.href = currentJob.url; link.download = detail.name; link.textContent = `Save ${detail.name}`; currentJob.row.append(link); finishJob('Ready to save');
  } else if (name === 'uploaded') { finishJob('Complete'); transfer?.list(folder,input('show-hidden').checked); }
  fileControls();
}
function connectFiles() {
  fileConn?.close(); transfer?.close();
  $('files-status').textContent = 'Connecting file transfer…';
  $('file-list').replaceChildren(); remoteEntries = []; selections.remote.clear(); remoteLoaded = false; remoteHistory = []; $('files-password-form').hidden = true;
  const client = new Connection(true);
  fileConn = client;
  const engine = new FileTransferClient(
    message => { if (!client.authenticated || !client.permissions.file || client !== fileConn) throw new Error('File connection is unavailable'); client._ws!.sendMessage(message); },
    (name, detail) => { if (client === fileConn) fileEvent(name, detail); },
    () => client._ws!.drain(),
  );
  transfer = engine;
  client.onFileResponse = response => engine.handle(response);
  client.onFileAction = action => { void engine.handleAction(action); return Promise.resolve(); };
  fileControls(); void client.start(sessionConfig.id);
}
button('files').addEventListener('click', () => { $<HTMLDialogElement>('files-dialog').showModal(); if (!fileConn?.authenticated) connectFiles(); });
button('retry-files').addEventListener('click', connectFiles);
window.addEventListener('rustdesk-files-status', ((e: CustomEvent) => {
  if (e.detail.connection !== fileConn) return;
  const {type, title, text} = e.detail;
  $('files-status').textContent = text || title;
  const form = $('files-password-form'); form.hidden = !['input-password','re-input-password'].includes(type);
  if (!form.hidden) input('files-password').focus();
  if (type === 'error') { fileConn?.close(); transfer?.close(); }
  fileControls();
}) as EventListener);
window.addEventListener('rustdesk-files-peer_info', ((e: CustomEvent) => { if (e.detail.connection !== fileConn) return; $('files-password-form').hidden = true; fileControls(); transfer?.list(''); }) as EventListener);
window.addEventListener('rustdesk-files-permission', ((e: CustomEvent) => { if (e.detail.connection !== fileConn) return; if (!fileConn?.permissions.file) { transfer?.cancel(); fileConn?.close(); } fileControls(); }) as EventListener);
window.addEventListener('rustdesk-files-closed', ((e: CustomEvent) => { if (e.detail.connection !== fileConn) return; transfer?.close(); $('file-progress').hidden = true; if(currentJob) finishJob('Connection closed'); for(const job of queue) if(job.state === 'Queued') {job.state = 'Cancelled'; job.status.textContent = job.state;} fileControls(); }) as EventListener);
$('files-password-form').addEventListener('submit', e => { e.preventDefault(); const value = input('files-password').value; input('files-password').value = ''; $('files-password-form').hidden = true; fileConn?.login(value); });
$('directory-form').addEventListener('submit', e => { e.preventDefault(); fileAction(() => transfer?.list(input('remote-path').value,input('show-hidden').checked)); });
button('files-home').addEventListener('click', () => fileAction(() => transfer?.list('',input('show-hidden').checked)));
button('files-refresh').addEventListener('click', () => fileAction(() => transfer?.list(folder,input('show-hidden').checked)));
input('show-hidden').addEventListener('change', () => fileAction(() => transfer?.list(folder,input('show-hidden').checked)));
button('files-up').addEventListener('click', () => fileAction(() => {
  const path = folder.replace(/[\\/]+$/,''); const index = Math.max(path.lastIndexOf('/'),path.lastIndexOf('\\'));
  const parent = index < 0 ? '' : index === 0 ? '/' : path.slice(0,index+1);
  transfer?.list(parent,input('show-hidden').checked);
}));
button('files-back').addEventListener('click', () => fileAction(() => { const path = remoteHistory.pop(); if(path !== undefined) {remoteBack = true; transfer?.list(path,input('show-hidden').checked);} }));
button('choose-files').addEventListener('click', () => input('upload-file').click());
button('choose-folder').addEventListener('click', async () => {
  try {
    if('showDirectoryPicker' in window) { const handle = await (window as any).showDirectoryPicker({mode:'read'}); localHandles = [handle]; localFolder = ''; localHistory = []; await loadLocal(); }
    else input('upload-folder').click();
  } catch(error:any) { if(error.name !== 'AbortError') $('files-status').textContent = error.message; }
});
for(const id of ['upload-file','upload-folder']) input(id).addEventListener('change', () => {
  localFiles = Array.from(input(id).files || []); localHandles = []; localFolder = ''; localHistory = []; input(id).value = ''; fileAction(loadLocal);
});
button('local-refresh').addEventListener('click', () => fileAction(loadLocal));
button('local-up').addEventListener('click', () => fileAction(async () => {localHistory.push({folder:localFolder,handles:[...localHandles]}); if(localHandles.length > 1) localHandles.pop(); else localFolder = localFolder.split('/').slice(0,-1).join('/'); await loadLocal();}));
button('local-home').addEventListener('click', () => fileAction(async () => {localHistory.push({folder:localFolder,handles:[...localHandles]}); localHandles = localHandles.slice(0,1); localFolder = ''; await loadLocal();}));
button('local-back').addEventListener('click', () => fileAction(async () => {const previous = localHistory.pop(); if(previous) {localFolder = previous.folder; localHandles = previous.handles; await loadLocal();}}));
for(const side of ['local','remote'] as const) input(`${side}-search`).addEventListener('input', () => renderEntries(side));
for(const el of document.querySelectorAll<HTMLButtonElement>('[data-sort]')) el.addEventListener('click', () => {
  const [side,key] = el.dataset.sort!.split(':') as ['local'|'remote',string]; const sort = sorting[side]; sort.descending = sort.key === key ? !sort.descending : false; sort.key = key;
  for(const header of el.closest('tr')!.querySelectorAll('th')) header.removeAttribute('aria-sort'); el.closest('th')!.setAttribute('aria-sort',sort.descending ? 'descending':'ascending'); renderEntries(side);
});
button('send-files').addEventListener('click', () => enqueue('upload'));
button('receive-files').addEventListener('click', () => enqueue('download'));
button('cancel-transfer').addEventListener('click', () => fileAction(() => { for(const job of queue) if(job.state === 'Queued') {job.state = 'Cancelled'; job.status.textContent = job.state;} transfer?.cancel(); }));
button('clear-transfers').addEventListener('click', () => {queue = queue.filter(job => {if(job === currentJob || job.state === 'Queued') return true; if(job.url) URL.revokeObjectURL(job.url); job.row.remove(); return false;}); $('queue-empty').hidden = !!queue.length;});
window.addEventListener('beforeunload', () => {for(const job of queue) if(job.url) URL.revokeObjectURL(job.url);});
fileControls();
async function start() {
  const response = await fetch('/webclient/session-config', {credentials:'same-origin',cache:'no-store'});
  if (!response.ok) throw new Error('This access session has expired or is unavailable. Open a new link.');
  configureSession(await response.json());
  document.querySelector('#target')!.textContent = `Remote device ${sessionConfig.id}`; $('files-peer').textContent = sessionConfig.id;
  await ready; conn = new Connection(); button("files").disabled = false; await conn.start(sessionConfig.id);
}
button("files").disabled = true;
start().catch(e => stop(e.message));
