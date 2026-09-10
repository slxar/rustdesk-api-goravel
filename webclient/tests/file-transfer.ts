import assert from 'node:assert/strict';
import { File } from 'node:buffer';
import FileTransfer, { MAX_FILE_SIZE, fileBasename } from '../src/file-transfer';
import { FileResponse, FileAction, FileType, Message } from '../src/message';
import { PeerInfo, Misc, PermissionInfo_Permission, ControlKey } from '../src/message';
import Connection from '../src/connection';
import Websock from '../src/websock';
import sodium from 'libsodium-wrappers';

function setup(drain = async () => {}) {
  const sent: any[] = [], events: { name: string; data: any }[] = [];
  const transfer = new FileTransfer(message => {
    // Exercise the real wire serializer, including the new total_size field.
    sent.push(Message.decode(Message.encode(Message.fromPartial(message)).finish()));
  }, (name, data) => events.push({ name, data }), drain);
  return { transfer, sent, events,
    response: (value: any) => transfer.handle(FileResponse.fromPartial(value)),
    action: (value: any) => transfer.handleAction(FileAction.fromPartial(value)),
  };
}

const bytes = new TextEncoder().encode('hello');
async function preparedDownload(size = bytes.length) {
  const h = setup();
  h.transfer.download('/tmp/example.txt');
  await h.response({ dir: { id: 1, path: '/tmp/example.txt', entries: [{ name: '', entry_type: FileType.File, size }] } });
  await h.response({ digest: { id: 1, file_num: 0, file_size: size } });
  assert.equal(h.sent.at(-1).file_action.send_confirm.offset_blk, 0);
  return h;
}
for (const name of ['', '.', '..', 'a\0b', 'C:']) assert.throws(() => fileBasename(name));
assert.equal(fileBasename('C:\\Users\\example.txt'), 'example.txt');

{
  const h = await preparedDownload();
  await h.response({ block: { id: 1, file_num: 0, data: bytes } });
  await h.response({ block: { id: 1, file_num: 0, data: new Uint8Array() } });
  await h.response({ done: { id: 1, file_num: 1 } });
  assert.equal(await h.events.find(e => e.name === 'download')!.data.blob.text(), 'hello');
  assert.equal(h.transfer.busy, false);
}
{
  const h = await preparedDownload(0);
  await h.response({ block: { id: 1, file_num: 0, data: new Uint8Array() } });
  await h.response({ done: { id: 1, file_num: 1 } });
  assert.equal(h.events.find(e => e.name === 'download')!.data.blob.size, 0);
}
{
  const h = await preparedDownload();
  // Standard Zstd frame containing one uncompressed block, decoded via wasm.
  const zstd = new Uint8Array([0x28, 0xb5, 0x2f, 0xfd, 0x20, 5, 0x29, 0, 0, ...bytes]);
  await h.response({ block: { id: 1, file_num: 0, data: zstd, compressed: true } });
  await h.response({ done: { id: 1, file_num: 1 } });
  assert.equal(await h.events.find(e => e.name === 'download')!.data.blob.text(), 'hello');
}
for (const block of [
  { file_num: 1, data: bytes }, { file_num: 0, blk_id: 1, data: bytes },
  { file_num: 0, data: new Uint8Array(6) },
  { file_num: 0, compressed: true, data: new Uint8Array([1, 2, 3]) },
]) {
  const h = await preparedDownload();
  await h.response({ block: { id: 1, ...block } });
  assert.equal(h.transfer.busy, false);
  assert.equal(h.sent.at(-1).file_action.cancel.id, 1);
  assert.ok(!h.events.some(e => e.name === 'download'));
}
{
  const h = await preparedDownload();
  let resume!: () => void;
  (h.transfer as any).decoder = { init: () => new Promise<void>(resolve => { resume = resolve; }), decode: () => { throw new Error('stale'); } };
  const incoming = h.response({ block: { id: 1, file_num: 0, compressed: true, data: bytes } });
  h.transfer.cancel(); h.transfer.download('/tmp/new-job.txt');
  resume(); await incoming;
  assert.equal(h.transfer.busy, true, 'A cancelled decoder must not abort its replacement job');
  h.transfer.close();
}
{
  const h = await preparedDownload();
  await h.response({ done: { id: 1, file_num: 1 } });
  assert.equal(h.transfer.busy, false, 'Short downloads must fail');
  assert.ok(!h.events.some(e => e.name === 'download'));
}
for (const entries of [
  [{ name: '', entry_type: FileType.File, size: MAX_FILE_SIZE + 1 }],
  [{ name: '../escape', entry_type: FileType.File, size: 1 }],
  [{ name: 'link', entry_type: FileType.FileLink, size: 1 }],
  [{ name: 'a', entry_type: FileType.File, size: 1 }, { name: 'b', entry_type: FileType.File, size: 1 }],
]) {
  const h = setup(); h.transfer.download('/tmp/test');
  await h.response({ dir: { id: 1, entries } });
  assert.equal(h.transfer.busy, false);
  assert.equal(h.sent.at(-1).file_action.cancel.id, 1);
}

for (const content of [new Uint8Array(), new Uint8Array(256 * 1024 + 5).fill(7)]) {
  const h = setup();
  h.transfer.upload('/tmp', new File([content], 'upload.txt', { lastModified: 1000 }) as any);
  assert.equal(h.sent[0].file_action.receive.total_size, content.length);
  assert.equal(h.sent.length, 2, 'Wait for host acceptance before writing');
  await h.action({ send_confirm: { id: 1, file_num: 0, offset_blk: 0 } });
  const blocks = h.sent.filter(m => m.file_response?.block).map(m => m.file_response.block.data);
  assert.ok(blocks.length >= 1, 'Empty uploads require an empty block');
  assert.deepEqual(Buffer.concat(blocks), Buffer.from(content));
  assert.equal(h.sent.at(-1).file_response.done.file_num, 1);
  assert.equal(h.transfer.busy, true, 'Wait for actual host completion');
  await h.response({ done: { id: 1, file_num: 1 } });
  assert.equal(h.transfer.busy, false);
  assert.ok(h.events.some(e => e.name === 'uploaded'));
}
for (const rejection of ['digest', 'skip', 'offset']) {
  const h = setup(); h.transfer.upload('/tmp', new File([bytes], 'test.txt') as any);
  if (rejection === 'digest') await h.response({ digest: { id: 1, file_num: 0, file_size: 3, is_upload: true } });
  else await h.action({ send_confirm: { id: 1, file_num: 0, ...(rejection === 'skip' ? { skip: true } : { offset_blk: 4 }) } });
  assert.equal(h.transfer.busy, false);
  assert.ok(!h.sent.some(m => m.file_response?.block), 'Never overwrite or resume automatically');
}
{
  let resume!: () => void;
  const h = setup(() => new Promise<void>(resolve => { resume = resolve; }));
  h.transfer.upload('/tmp', new File([bytes], 'test.txt') as any);
  const upload = h.action({ send_confirm: { id: 1, file_num: 0, offset_blk: 0 } });
  h.transfer.cancel(); resume(); await upload;
  assert.ok(!h.sent.some(m => m.file_response?.block), 'Cancelled uploads stop after backpressure wait');
}
{
  const h = setup(); h.transfer.download('/tmp/example');
  assert.throws(() => h.transfer.download('/tmp/another'));
  await h.response({ error: { id: 99, error: 'stale' } });
  assert.equal(h.transfer.busy, true, 'Unrelated errors must not cancel current file');
  await h.response({ error: { id: 1, error: 'Permission denied' } });
  assert.equal(h.transfer.busy, false);
  assert.equal(h.events.at(-1)!.data.text, 'Permission denied');
}
{
  const original = globalThis.setTimeout;
  let expire!: () => void;
  globalThis.setTimeout = ((callback: () => void) => { expire = callback; return 0; }) as any;
  const h = setup();
  try { h.transfer.download('/tmp/stalled'); } finally { globalThis.setTimeout = original; }
  expire();
  assert.equal(h.transfer.busy, false);
  assert.equal(h.sent.at(-1).file_action.cancel.id, 1);
}
for (const direction of ['download', 'upload']) {
  const events: any[] = [];
  const transfer = new FileTransfer(() => { throw new Error('Socket closed'); }, (name, data) => events.push({ name, data }), async () => {});
  if (direction === 'download') transfer.download('/tmp/file.txt');
  else transfer.upload('/tmp', new File([bytes], 'file.txt') as any);
  assert.equal(transfer.busy, false, 'Initial send failure must release the queue immediately');
  assert.equal(events.at(-1).data.text, 'Socket closed');
}
console.log('File-transfer check passed: wire round trips, upload/download/empty/zstd, confirmation, limits, cancellation, errors and timeout.');

// Exercise the connection boundary as well as transfer messages: a file login
// must never silently become a desktop login or omit overwrite negotiation.
(globalThis as any).window = new EventTarget();
{
  const conn = new Connection(true), sent: any[] = [];
  conn._id = '123456789';
  conn._ws = { sendMessage: (value: any) => sent.push(Message.decode(Message.encode(Message.fromPartial(value)).finish())) } as any;
  conn._sendLoginMessage(bytes);
  const login = sent[0].login_request;
  assert.equal(login.username, '123456789');
  assert.equal(login.version, '1.4.9');
  assert.deepEqual(login.file_transfer, { dir: '', show_hidden: false });
  assert.equal(login.option, undefined);
  assert.equal(login.video_ack_required, false);
  const events: any[] = [];
  window.addEventListener('rustdesk-files-peer_info', e => events.push(e));
  conn.handlePeerInfo(PeerInfo.fromPartial({ platform: 'Windows', displays: [] }));
  assert.equal(conn.authenticated, true, 'File sessions work without displays');
  assert.equal(events.length, 1);
  assert.equal(conn.handleMisc(Misc.fromPartial({ permission_info: { permission: PermissionInfo_Permission.File, enabled: false } })), true);
  assert.equal(conn.permissions.file, false);
  assert.equal(conn.handleMisc(Misc.fromPartial({ permission_info: { permission: 999 as any, enabled: true } })), true, 'Future permissions must not stop receiving');
}
{
  const conn = new Connection(), sent: any[] = [];
  conn._options = {};
  conn._ws = { sendMessage: (value: any) => sent.push(value) } as any;
  const keys = () => { conn.inputKey('a', true, false, false, false, false, false); conn.inputMouse(); conn.inputString('hello'); conn.lockScreen(); conn.ctrlAltDel(); };
  keys(); assert.equal(sent.length, 0, 'No input before host login');
  conn.authenticated = true; conn.permissions.keyboard = false;
  keys(); assert.equal(sent.length, 0, 'Host denied keyboard controls');
  conn.permissions.keyboard = true;
  conn._peerInfo = PeerInfo.fromPartial({ platform: 'Windows', sas_enabled: true });
  keys(); assert.equal(sent.length, 5);
  assert.equal(sent.at(-1).key_event.control_key, ControlKey.CtrlAltDel);
  conn.handleMisc(Misc.fromPartial({ permission_info: { permission: PermissionInfo_Permission.BlockInput, enabled: false } }));
  const beforeBlock = sent.length;
  conn.toggleOption('block-input'); assert.equal(sent.length, beforeBlock, 'Host block-input permission is enforced');
  conn.permissions.blockInput = true; conn.toggleOption('block-input'); assert.equal(sent.length, beforeBlock + 1);
  conn._peerInfo.platform = 'Mac OS'; conn.toggleOption('block-input'); assert.equal(sent.length, beforeBlock + 1, 'Block-input is Windows-only');
  conn._peerInfo.platform = 'Android'; conn.toggleOption('lock-after-session-end'); assert.equal(sent.length, beforeBlock + 1, 'Android does not support lock-after');
  conn.permissions.clipboard = false;
  assert.throws(() => conn.sendClipboard('hello'));
  conn.permissions.clipboard = true;
  conn.sendClipboard('hello');
  assert.deepEqual(sent.at(-1).clipboard.content, bytes);
  assert.throws(() => conn.sendClipboard('x'.repeat(1024 * 1024 + 1)));
}
console.log('Connection contract check passed: dedicated file login, feature flags, permission updates, authenticated controls and clipboard bounds.');

{
  await sodium.ready;
  const key = sodium.crypto_secretbox_keygen();
  const socket = Object.create(Websock.prototype) as Websock;
  const frames: Uint8Array[] = [];
  socket._websocket = { send: (data: Uint8Array) => frames.push(data.slice()) } as any;
  socket.setSecretKey(key);
  for (let i = 0; i < 300; i++) {
    socket.sendMessage({ file_action: { receive: { id: i + 1, path: '/tmp', total_size: 131072, files: [{ name: 'upload.bin', size: 131072, modified_time: 1, entry_type: FileType.File }] } } });
    socket.sendMessage({ file_response: { digest: { id: i + 1, file_num: 0, file_size: 131072, is_upload: true } } });
  }
  socket.sendMessage({ file_response: { block: { id: 300, file_num: 0, data: new Uint8Array(131072).fill(7) } } });
  for (let i = 0; i < frames.length; i++) {
    const nonce = new Uint8Array(24);
    new DataView(nonce.buffer).setBigUint64(0, BigInt(i + 1), true);
    const clear = sodium.crypto_secretbox_open_easy(frames[i], nonce, key);
    const message = Message.decode(clear);
    if (i < 600) assert.equal(i % 2 ? message.file_response!.digest!.id : message.file_action!.receive!.id, Math.floor(i / 2) + 1);
    else assert.equal(message.file_response!.block!.data.length, 131072);
  }
}
console.log('Upload encryption burst check passed: 601 consecutive frames, native little-endian nonce and large file blocks.');
