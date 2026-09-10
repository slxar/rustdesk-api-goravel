import {build} from 'esbuild';
import {mkdtemp,writeFile,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {spawnSync} from 'node:child_process';
const dir = await mkdtemp(join(tmpdir(),'rustdesk-browser-test-'));
try {
  for (const suite of ['security', 'file-transfer', 'access-link']) {
    const result = await build({entryPoints:[`tests/${suite}.ts`],bundle:true,platform:'node',format:'esm',write:false,banner:{js:"import {createRequire} from 'node:module'; const require = createRequire(import.meta.url);"}});
    const path = join(dir,`${suite}.mjs`); await writeFile(path,result.outputFiles[0].contents);
    const run = spawnSync(process.execPath,[path],{stdio:'inherit'});
    if (run.status !== 0) { process.exitCode=run.status ?? 1; break; }
  }
} finally { await rm(dir,{recursive:true,force:true}); }
