import {readdir,readFile,writeFile,copyFile} from 'node:fs/promises';
import {join} from 'node:path';
import {execFileSync} from 'node:child_process';
let notices='Third-party dependency license texts\n\n';
async function scan(dir) {
  for(const entry of await readdir(dir,{withFileTypes:true})) {
    const path=join(dir,entry.name);
    if(entry.isDirectory()) { if(entry.name!=='.bin') await scan(path); }
    else if(/^(license|licence|copying|notice)(\.|$)/i.test(entry.name)) notices+='\n--- '+path+' ---\n'+await readFile(path,'utf8')+'\n';
  }
}
await scan('node_modules');
notices=notices.split(/\r?\n/).map(line=>line.trimEnd()).join('\n').trimEnd()+'\n';
await writeFile('THIRD_PARTY_NOTICES.txt',notices);
for(const name of ['LICENSE','NOTICE','THIRD_PARTY_NOTICES.txt']) await copyFile(name,'../resources/webclient/'+name);
execFileSync('tar',['-czf','../resources/webclient/source.tar.gz','--exclude=node_modules','--exclude=.git','--exclude=.gitignore','--exclude=.DS_Store','.']);
