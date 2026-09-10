import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {runInNewContext} from 'node:vm';
const source = readFileSync('../http/controller/web/webclient.go','utf8');
const script = source.match(/const webOpenScript = `([^`]+)`/)![1];
async function open(failCleanup = false) {
  const calls: string[] = [];
  const button: any = {};
  const status: any = {};
  const registration = (scope: string) => ({scope,unregister: async () => { calls.push(scope); if (failCleanup) throw new Error('Cleanup unavailable'); return true; }});
  runInNewContext(script, {
    URL, Date: {now: () => 123},
    location: {origin:'https://example.test',hash:'#'+'A'.repeat(43),replace:(path: string) => calls.push(path)},
    history: {replaceState:() => calls.push('clear-fragment')},
    document: {getElementById:(id: string) => id === 'open' ? button : status},
    navigator: {serviceWorker: {getRegistrations:async () => [registration('https://example.test/webclient/'),registration('https://example.test/_admin/'),registration('https://other.test/webclient/')] }},
    fetch: async (path: string) => {calls.push(path);return {ok:true};},
  });
  await button.onclick();
  return {calls,status};
}
assert.deepEqual((await open()).calls,['clear-fragment','https://example.test/webclient/','/webclient/redeem','/webclient/?session=123']);
const failed = await open(true);
assert.deepEqual(failed.calls,['clear-fragment','https://example.test/webclient/']);
assert.equal(failed.status.textContent,'Cleanup unavailable');
console.log('Access-link check passed: retire only same-origin webclient workers before redemption; failure does not consume the link.');
