import sodium from 'libsodium-wrappers';
/** @type {Record<string, any>} */
const peers = {};
export const ready = sodium.ready;
export const getPeers = () => peers;
export const isDesktop = () => true;
export function msgbox(type, title, text) { pushEvent('status', {type, title, text}); }
export function pushEvent(name, detail) { window.dispatchEvent(new CustomEvent('rustdesk-' + name, {detail})); }
export function draw(frame) { pushEvent('frame', frame); }
export async function verify(signed, key) {
  await sodium.ready;
  if (typeof key === 'string') key = sodium.from_base64(key, sodium.base64_variants.ORIGINAL);
  return sodium.crypto_sign_open(signed, key);
}
export function genBoxKeyPair() { const p = sodium.crypto_box_keypair(); return [p.privateKey, p.publicKey]; }
export const genSecretKey = () => sodium.crypto_secretbox_keygen();
export const seal = (data, key, secret) => sodium.crypto_box_easy(data, new Uint8Array(24), key, secret);
function nonce(value) {
  const bytes = new Uint8Array(24);
  for (let i = 0; value > 0; i++, value = Math.floor(value / 256)) bytes[i] = value & 255;
  return bytes;
}
export const encrypt = (data, counter, key) => sodium.crypto_secretbox_easy(data, nonce(counter), key);
export const decrypt = (data, counter, key) => sodium.crypto_secretbox_open_easy(data, nonce(counter), key);
export const initAudio = (_channels, _sampleRate) => {};
export const playAudio = (_data) => {};
