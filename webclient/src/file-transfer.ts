import { ZSTDDecoder } from 'zstddec';
import { DeepPartial, Message, FileAction, FileResponse, FileType } from './message';

export const MAX_FILE_SIZE = 64 * 1024 * 1024;
const BLOCK_SIZE = 128 * 1024;
const TIMEOUT = 30_000;

export function fileBasename(path: string): string {
  const name = path.split(/[\\/]/).pop() || '';
  if (!name || name === '.' || name === '..' || /[\x00-\x1f\x7f:]/.test(name)) throw new Error('Invalid file name');
  return name;
}

function checkPath(path: string) {
  if (typeof path !== 'string' || path.length > 4096 || /[\x00-\x1f\x7f]/.test(path)) throw new Error('Invalid remote path');
}

type Job = {
  id: number; kind: 'upload' | 'download'; name: string; size: number;
  transferred: number; parts: Uint8Array[]; file?: File;
  stage: 'directory' | 'confirm' | 'stream' | 'done';
};

// ponytail: one file at a time, up to 64 MiB buffered for downloads. Use a
// browser filesystem stream before raising the limit or adding directory jobs.
export default class FileTransfer {
  private job?: Job;
  private nextID = 1;
  private timer?: ReturnType<typeof setTimeout>;
  private decoder = new ZSTDDecoder();
  constructor(
    private send: (message: DeepPartial<Message>) => void,
    private emit: (event: string, detail: any) => void,
    private drain: () => Promise<void>,
  ) {}

  get busy() { return !!this.job; }

  list(path: string, include_hidden = false) {
    checkPath(path);
    this.send({ file_action: { read_dir: { path, include_hidden } } });
  }

  download(path: string) {
    checkPath(path);
    const name = fileBasename(path);
    const job = this.begin('download', name, -1);
    try {
      this.send({ file_action: { send: { id: job.id, path, file_num: 0, include_hidden: false } } });
    } catch (error) { this.fail(error instanceof Error ? error.message : 'Download failed'); }
  }

  upload(path: string, file: File) {
    checkPath(path);
    if (fileBasename(file.name) !== file.name) throw new Error('Invalid file name');
    this.checkSize(file.size);
    const job = this.begin('upload', file.name, file.size);
    job.file = file;
    const modified = Math.floor(file.lastModified / 1000);
    try {
      this.send({ file_action: { receive: {
        id: job.id, path, file_num: 0, total_size: file.size,
        files: [{ name: file.name, entry_type: FileType.File, size: file.size, modified_time: modified }],
      } } });
      // The host confirms absent files, or sends a digest for an existing file.
      // No file contents are sent until this check completes.
      this.send({ file_response: { digest: { id: job.id, file_num: 0,
        file_size: file.size, last_modified: modified, is_upload: true } } });
    } catch (error) { this.fail(error instanceof Error ? error.message : 'Upload failed'); }
  }

  private begin(kind: Job['kind'], name: string, size: number) {
    if (this.job) throw new Error('Finish or cancel the current transfer first');
    const job: Job = { id: this.nextID++, kind, name, size, transferred: 0, parts: [], stage: kind === 'download' ? 'directory' : 'confirm' };
    this.job = job;
    this.touch();
    this.emit('status', { text: kind === 'download' ? 'Preparing download…' : 'Checking destination…', busy: true });
    return job;
  }

  private touch() {
    clearTimeout(this.timer);
    this.timer = setTimeout(() => this.fail('File transfer timed out'), TIMEOUT);
  }

  private checkSize(size: number) {
    if (!Number.isSafeInteger(size) || size < 0 || size > MAX_FILE_SIZE) throw new Error('Files must be 64 MiB or smaller');
  }

  async handle(response: FileResponse) {
    const job = this.job;
    try {
      if (response.dir) {
        const dir = response.dir;
        if (dir.id === 0) { this.emit('directory', dir); return; }
        if (!job || dir.id !== job.id || job.kind !== 'download') return;
        if (job.stage !== 'directory' || dir.entries.length !== 1) throw new Error('Select one regular file to download');
        const file = dir.entries[0];
        if (file.entry_type !== FileType.File || (file.name && fileBasename(file.name) !== file.name)) throw new Error('Select one regular file to download');
        this.checkSize(file.size);
        job.size = file.size;
        job.stage = 'confirm';
        this.touch();
      } else if (response.error) {
        if (response.error.id === 0) this.emit('status', { text: response.error.error, error: true, busy: this.busy });
        else if (job && response.error.id === job.id) this.fail(response.error.error || 'File transfer failed');
      } else if (response.digest) {
        const digest = response.digest;
        if (!job || digest.id !== job.id) return;
        if (job.kind === 'upload') throw new Error('A file already exists at the destination. Rename the local file before uploading.');
        if (job.stage !== 'confirm' || digest.file_num !== 0 || digest.is_upload || digest.file_size !== job.size) throw new Error('Invalid download confirmation');
        job.stage = 'stream';
        this.send({ file_action: { send_confirm: { id: job.id, file_num: 0, offset_blk: 0 } } });
        this.touch();
      } else if (response.block) {
        const block = response.block;
        if (!job || block.id !== job.id) return;
        if (job.kind !== 'download' || job.stage !== 'stream' || block.file_num !== 0 || block.blk_id !== 0 || block.data.length > BLOCK_SIZE) throw new Error('Invalid file block');
        let data = block.data;
        if (block.compressed) {
          await this.decoder.init();
          if (this.job !== job) return;
          data = this.decoder.decode(data, BLOCK_SIZE);
          if (data.length === 0 || data.length > BLOCK_SIZE) throw new Error('Invalid compressed file block');
        }
        if (this.job !== job) return;
        if (job.transferred + data.length > job.size) throw new Error('File exceeds its announced size');
        if (data.length) job.parts.push(data.slice());
        job.transferred += data.length;
        this.touch();
        this.progress(job);
      } else if (response.done) {
        const done = response.done;
        if (!job || done.id !== job.id) return;
        if (done.file_num !== 1 || job.transferred !== job.size || !['stream', 'done'].includes(job.stage)) throw new Error('Incomplete file transfer');
        if (job.kind === 'upload' && job.stage !== 'done') throw new Error('Unexpected upload completion');
        const blob = job.kind === 'download' ? new Blob(job.parts as BlobPart[], { type: 'application/octet-stream' }) : undefined;
        this.clear();
        this.emit(job.kind === 'download' ? 'download' : 'uploaded', { name: job.name, blob, size: job.size });
        this.emit('status', { text: job.kind === 'download' ? 'Download ready.' : 'Upload complete.', busy: false });
      }
    } catch (error) { if (this.job === job) this.fail(error instanceof Error ? error.message : 'File transfer failed'); }
  }

  async handleAction(action: FileAction) {
    const confirm = action.send_confirm;
    const job = this.job;
    if (!confirm || !job || confirm.id !== job.id || job.kind !== 'upload') return;
    try {
      if (job.stage !== 'confirm' || confirm.file_num !== 0 || confirm.skip || confirm.offset_blk !== 0) throw new Error('Destination already exists or upload was rejected');
      job.stage = 'stream';
      this.touch();
      // Even an empty file needs an empty block to create it on the host.
      do {
        await this.drain();
        if (this.job !== job) return;
        const data = new Uint8Array(await job.file!.slice(job.transferred, job.transferred + BLOCK_SIZE).arrayBuffer());
        if (this.job !== job) return;
        if (!data.length && job.transferred < job.size) throw new Error('Could not read local file');
        this.send({ file_response: { block: { id: job.id, file_num: 0, data, compressed: false, blk_id: 0 } } });
        job.transferred += data.length;
        this.touch();
        this.progress(job);
      } while (job.transferred < job.size);
      await this.drain();
      if (this.job !== job) return;
      job.stage = 'done';
      this.send({ file_response: { done: { id: job.id, file_num: 1 } } });
      this.touch();
    } catch (error) { if (this.job === job) this.fail(error instanceof Error ? error.message : 'Upload failed'); }
  }

  private progress(job: Job) { this.emit('progress', { name: job.name, transferred: job.transferred, total: job.size, direction: job.kind }); }
  private clear() { clearTimeout(this.timer); this.timer = undefined; this.job = undefined; }
  private fail(text: string) {
    const job = this.job;
    this.clear();
    if (job) { try { this.send({ file_action: { cancel: { id: job.id } } }); } catch {} }
    this.emit('status', { text, error: true, busy: false });
  }
  cancel() { if (this.job) this.fail('Transfer cancelled.'); }
  close() { this.clear(); }
}
