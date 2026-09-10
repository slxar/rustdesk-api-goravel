import * as message from "./message.js";
import * as rendezvous from "./rendezvous.js";
import * as globals from "./globals";

type Keys = "message" | "open" | "close" | "error";

export default class Websock {
  _websocket: WebSocket;
  _eventHandlers: { [key in Keys]: Function };
  _buf: (rendezvous.RendezvousMessage | message.Message)[];
  _status: any;
  _latency: number;
  _secretKey: [Uint8Array, number, number] | undefined;
  _closing = false;
  _uri: string;
  _isRendezvous: boolean;

  constructor(uri: string, isRendezvous: boolean = true) {
    this._eventHandlers = {
      message: (_: any) => {},
      open: () => {},
      close: () => {},
      error: () => {},
    };
    this._uri = uri;
    this._status = "";
    this._buf = [];
    this._websocket = new WebSocket(uri);
    this._websocket.onmessage = this._recv_message.bind(this);
    this._websocket.binaryType = "arraybuffer";
    this._latency = new Date().getTime();
    this._isRendezvous = isRendezvous;
  }

  latency(): number {
    return this._latency;
  }

  setSecretKey(key: Uint8Array) {
    this._secretKey = [key, 0, 0];
  }

  sendMessage(json: message.DeepPartial<message.Message>) {
    let data = message.Message.encode(
      message.Message.fromPartial(json)
    ).finish();
    let k = this._secretKey;
    if (k) {
      k[1] += 1;
      data = globals.encrypt(data, k[1], k[0]);
    }
    this._websocket.send(data as Uint8Array<ArrayBuffer>);
  }

  async drain() {
    const deadline = Date.now() + 30000;
    while (this._websocket.bufferedAmount > 512 * 1024) {
      if (this._status !== 'open' || Date.now() > deadline) throw new Error('File transfer connection stalled');
      await new Promise(resolve => setTimeout(resolve, 20));
    }
    if (this._status !== 'open') throw new Error('Connection closed');
  }

  sendRendezvous(data: rendezvous.DeepPartial<rendezvous.RendezvousMessage>) {
    this._websocket.send(
      rendezvous.RendezvousMessage.encode(
        rendezvous.RendezvousMessage.fromPartial(data)
      ).finish() as Uint8Array<ArrayBuffer>
    );
  }

  parseMessage(data: Uint8Array) {
    return message.Message.decode(data);
  }

  parseRendezvous(data: Uint8Array) {
    return rendezvous.RendezvousMessage.decode(data);
  }

  // Event Handlers
  off(evt: Keys) {
    this._eventHandlers[evt] = () => {};
  }

  on(evt: Keys, handler: Function) {
    this._eventHandlers[evt] = handler;
  }

  async open(timeout: number = 12000): Promise<Websock> {
    return new Promise((resolve, reject) => {
      setTimeout(() => {
        if (this._status != "open") {
          reject(this._status || "Timeout");
        }
      }, timeout);
      this._websocket.onopen = () => {
        this._latency = new Date().getTime() - this._latency;
        this._status = "open";
        console.debug(">> WebSock.onopen");
        if (this._websocket?.protocol) {
          console.info(
            "Server choose sub-protocol: " + this._websocket.protocol
          );
        }

        this._eventHandlers.open();
        console.info("WebSock.onopen");
        resolve(this);
      };
      this._websocket.onclose = (e) => {
        if (this._status == "open") {
          // e.code 1000 means that the connection was closed normally.
          //
        }
        this._status = e;
        if (!this._closing) console.error("WebSocket closed unexpectedly:", e);
        this._eventHandlers.close(e);
        reject("Reset by the peer");
      };
      this._websocket.onerror = (e: any) => {
        if (!this._status) {
          reject("Failed to connect to " + (this._isRendezvous ? "rendezvous" : "relay") + " server");
          return;
        }
        this._status = e;
        console.error("WebSock.onerror: ")
        console.error(e);
        this._eventHandlers.error(e);
      };
    });
  }

  async next(
    timeout = 12000
  ): Promise<rendezvous.RendezvousMessage | message.Message> {
    const func = (
      resolve: (value: rendezvous.RendezvousMessage | message.Message) => void,
      reject: (reason: any) => void,
      tm0: number
    ) => {
      if (this._buf.length) {
        resolve(this._buf[0]);
        this._buf.splice(0, 1);
      } else {
        if (this._status != "open") {
          reject(this._status);
          return;
        }
        if (timeout > 0 && new Date().getTime() > tm0 + timeout) {
          reject("Timeout");
        } else {
          setTimeout(() => func(resolve, reject, tm0), 1);
        }
      }
    };
    return new Promise((resolve, reject) => {
      func(resolve, reject, new Date().getTime());
    });
  }

  close() {
    this._closing = true;
    this._status = "";
    if (this._websocket) {
      if (
        this._websocket.readyState === WebSocket.OPEN ||
        this._websocket.readyState === WebSocket.CONNECTING
      ) {
        console.info("Closing WebSocket connection");
        this._websocket.close();
      }

      this._websocket.onmessage = () => {};
    }
  }

  _recv_message(e: any) {
    try {
      if (!(e.data instanceof window.ArrayBuffer)) throw new Error("Invalid WebSocket frame type");
      let bytes = new Uint8Array(e.data);
      const key = this._secretKey;
      if (key) bytes = globals.decrypt(bytes, ++key[2], key[0]);
      this._buf.push(this._isRendezvous ? this.parseRendezvous(bytes) : this.parseMessage(bytes));
      this._eventHandlers.message(e.data);
    } catch (error) {
      this.close();
      this._status = error instanceof Error ? error : new Error("Invalid encrypted frame");
    }
  }
}
