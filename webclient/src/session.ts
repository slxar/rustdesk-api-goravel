export type SessionConfig = {id: string; key: string; id_ws: string; relay_ws: string};
export let sessionConfig: SessionConfig;
export function configureSession(value: SessionConfig) {
  if (!/^[A-Za-z0-9_-]{6,64}$/.test(value.id) || !value.key) throw new Error('Invalid browser session configuration');
  for (const field of ['id_ws', 'relay_ws'] as const) {
    const url = new URL(value[field], location.href);
    const scheme = location.protocol === 'https:' ? 'wss:' : 'ws:';
    if (value[field].startsWith('/')) url.protocol = scheme;
    if (url.host !== location.host || url.protocol !== scheme || !url.pathname.startsWith('/webclient/ws/')) {
      throw new Error('Untrusted WebSocket endpoint');
    }
    value[field] = url.href;
  }
  sessionConfig = Object.freeze(value);
}
