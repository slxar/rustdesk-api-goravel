import Websock from "./websock";
import * as message from "./message.js";
import * as rendezvous from "./rendezvous.js";
import { loadVp9 } from "./codec";
import * as sha256 from "fast-sha256";
import * as globals from "./globals";
import { decompress, mapKey, sleep } from "./common";

import {sessionConfig} from "./session";

type MsgboxCallback = (type: string, title: string, text: string) => void;
type DrawCallback = (data: Uint8Array) => void;
//const cursorCanvas = document.createElement("canvas");

export default class Connection {
  _clientID = "web-" + crypto.randomUUID();
  _msgs: any[];
  _ws: Websock | undefined;
  _interval: any;
  _id: string;
  _hash: message.Hash | undefined;
  _msgbox: MsgboxCallback;
  _draw: DrawCallback;
  _peerInfo: message.PeerInfo | undefined;
  _firstFrame: Boolean | undefined;
  _videoDecoder: any;
  _password: Uint8Array | undefined;
  _options: any;
  _videoTestSpeed: number[];
  //_cursors: { [name: number]: any };

  onFileAction?: (action: message.FileAction) => Promise<void>;
  onFileResponse?: (response: message.FileResponse) => Promise<void>;
  authenticated = false;
  private _closed = false;
  permissions = {keyboard: true, clipboard: true, file: true, blockInput: true};

  constructor(public fileTransfer = false) {
    console.info("RustDesk browser session client ID:", this._clientID);
    this._msgbox = (type, title, text) => this.emit("status", {type, title, text});
    this._draw = globals.draw;
    this._msgs = [];
    this._id = "";
    this._videoTestSpeed = [0, 0];
    //this._cursors = {};
  }

  emit(name: string, detail: any) { globals.pushEvent((this.fileTransfer ? "files-" : "") + name, {...detail, connection: this}); }

  async start(id: string) {
    this._closed = false;
    try {
      await this._start(id);
    } catch (e: any) {
      if (this._closed) return;
      this.close();
      this.msgbox(
        "error",
        "Connection Error",
        e.type == "close" ? "Reset by the peer" : String(e)
      );
    }
  }

  async _start(id: string) {
    if (!this._options) {
      this._options = globals.getPeers()[id] || {};
    }
    this.applyWebClientDefaults();
    if (!this._password) {
      const p = this.getOption("password");
      if (p) {
        try {
          this._password = Uint8Array.from(JSON.parse("[" + p + "]"));
        } catch (e) {
          console.error(e);
        }
      }
    }
    this._interval = setInterval(() => {
      while (this._msgs.length) {
        this._ws?.sendMessage(this._msgs[0]);
        this._msgs.splice(0, 1);
      }
    }, 1);
    if (!this.fileTransfer) await this.loadVideoDecoder();
    if (this._closed) return;
    const uri = sessionConfig.id_ws;
    const ws = new Websock(uri, true);
    this._ws = ws;
    if (id !== sessionConfig.id) throw new Error("Session target mismatch");
    this._id = id;
    console.log(
      new Date() + ": Conntecting to rendezvoous server: " + uri + ", for " + id
    );
    await ws.open();
    console.log(new Date() + ": Connected to rendezvoous server");
    const conn_type = this.fileTransfer ? rendezvous.ConnType.FILE_TRANSFER : rendezvous.ConnType.DEFAULT_CONN;
    const nat_type = rendezvous.NatType.SYMMETRIC;
    const punch_hole_request = rendezvous.PunchHoleRequest.fromPartial({
      id,
      licence_key: sessionConfig.key,
      conn_type,
      nat_type,

    });
    ws.sendRendezvous({ punch_hole_request });
    const msg = (await ws.next()) as rendezvous.RendezvousMessage;
    ws.close();
    console.log(new Date() + ": Got relay response", msg);
    const phr = msg.punch_hole_response;
    const rr = msg.relay_response;
    if (phr) {
      if (phr?.other_failure) {
        this.msgbox("error", "Error", phr?.other_failure);
        return;
      }
      if (phr.failure != rendezvous.PunchHoleResponse_Failure.UNRECOGNIZED) {
        switch (phr?.failure) {
          case rendezvous.PunchHoleResponse_Failure.ID_NOT_EXIST:
            this.msgbox("error", "Error", "ID does not exist");
            break;
          case rendezvous.PunchHoleResponse_Failure.OFFLINE:
            this.msgbox("error", "Error", "Remote desktop is offline");
            break;
          case rendezvous.PunchHoleResponse_Failure.LICENSE_MISMATCH:
            this.msgbox("error", "Error", "Key mismatch");
            break;
          case rendezvous.PunchHoleResponse_Failure.LICENSE_OVERUSE:
            this.msgbox("error", "Error", "Key overuse");
            break;
        }
      }
    } else if (rr) {
      if (!rr.version) {
        this.msgbox("error", "Error", "Remote version is low, not support web");
        return;
      }
      await this.connectRelay(rr);
    }
  }

  async connectRelay(rr: rendezvous.RelayResponse) {
    if (this._closed) return;
    const pk = rr.pk;
    const uri = sessionConfig.relay_ws;
    const uuid = rr.uuid;
    console.log(new Date() + ": Connecting to relay server: " + uri);
    const ws = new Websock(uri, false);
    this._ws = ws;
    await ws.open();
    console.log(new Date() + ": Connected to relay server");
    this._ws = ws;
    const request_relay = rendezvous.RequestRelay.fromPartial({
      id: this._id,
      licence_key: sessionConfig.key,
      uuid,
    });
    ws.sendRendezvous({ request_relay });
    const secure = (await this.secure(pk)) || false;
    this.emit("connection_ready", { secure, direct: false });
    await this.msgLoop();
  }

  async secure(signedPeerKey: Uint8Array | undefined) {
    if (!signedPeerKey?.length) throw new Error("Server did not provide a signed peer key");
    const peer = message.IdPk.decode(await globals.verify(signedPeerKey, sessionConfig.key));
    if (peer.id !== this._id || peer.pk.length !== 32) throw new Error("Invalid server-signed peer identity");
    const msg = (await this._ws?.next()) as message.Message;
    if (!msg?.signed_id?.id?.length) throw new Error("Peer did not provide a signed identity");
    const box = message.IdPk.decode(await globals.verify(msg.signed_id.id, peer.pk));
    if (box.id !== this._id || box.pk.length !== 32) throw new Error("Invalid signed peer identity");
    const [mySk, asymmetric_value] = globals.genBoxKeyPair();
    const secretKey = globals.genSecretKey();
    const symmetric_value = globals.seal(secretKey, box.pk, mySk);
    this._ws!.sendMessage({public_key: message.PublicKey.fromPartial({asymmetric_value, symmetric_value})});
    this._ws!.setSecretKey(secretKey);
    return true;
  }

  async msgLoop() {
    while (true) {
      const msg = (await this._ws?.next(0)) as message.Message;
      // console.log("msg", msg);
      if (msg?.hash) {
        this._hash = msg?.hash;
        const tmp = this.getOption('tmppwd')
        if(!this._password && tmp){
          this._password = Uint8Array.from(JSON.parse("[" + tmp + "]"));
          this.setOption('tmppwd', '')
        }
        if (!this._password)
          this.msgbox("input-password", "Password Required", "");
        this.login();
      } else if (msg?.test_delay) {
        const test_delay = msg?.test_delay;
        console.log(test_delay);
        if (!test_delay.from_client) {
          this._ws?.sendMessage({ test_delay });
        }
      } else if (msg?.login_response) {
        const r = msg?.login_response;
        if (r.error) {
          if (r.error == "Wrong Password") {
            this._password = undefined;
            this.msgbox(
              "re-input-password",
              r.error,
              "Do you want to enter again?"
            );
          } else {
            this.msgbox("error", "Login Error", r.error);
          }
        } else if (r.peer_info) {
          this.handlePeerInfo(r.peer_info);
        }
      } else if (msg?.file_action && this.fileTransfer) {
        await this.onFileAction?.(msg.file_action);
      } else if (msg?.file_response && this.fileTransfer) {
        await this.onFileResponse?.(msg.file_response);
      } else if (msg?.video_frame) {
        this.handleVideoFrame(msg?.video_frame!);
      } else if (msg?.clipboard) {
        const cb = msg?.clipboard;
        if (!this.permissions.clipboard || cb.content.length > 1024 * 1024) continue;
        if (cb.compress) {
          const c = await decompress(cb.content);
          if (!c) continue;
          if (c.length > 1024 * 1024) continue;
          cb.content = c;
        }
        try {
          if (this.permissions.clipboard) this.emit("clipboard", {text: new TextDecoder().decode(cb.content)});
        } catch (e) {
          console.error(e);
        }
        // this.emit("clipboard", cb);
      } else if (msg?.cursor_data) {
        const cd = msg?.cursor_data;
        const c = await decompress(cd.colors);
        if (!c) continue;
        cd.colors = c;
        this.emit("cursor_data", cd);
        /*
        let ctx = cursorCanvas.getContext("2d");
        cursorCanvas.width = cd.width;
        cursorCanvas.height = cd.height;
        let imgData = new ImageData(
          new Uint8ClampedArray(c),
          cd.width,
          cd.height
        );
        ctx?.clearRect(0, 0, cd.width, cd.height);
        ctx?.putImageData(imgData, 0, 0);
        let url = cursorCanvas.toDataURL();
        const img = document.createElement("img");
        img.src = url;
        this._cursors[cd.id] = img;
        //cursorCanvas.width /= 2.;
        //cursorCanvas.height /= 2.;
        //ctx?.drawImage(img, cursorCanvas.width, cursorCanvas.height);
        url = cursorCanvas.toDataURL();
        document.body.style.cursor =
          "url(" + url + ")" + cd.hotx + " " + cd.hoty + ", default";
        console.log(document.body.style.cursor);
        */
      } else if (msg?.cursor_id) {
        this.emit("cursor_id", { id: msg?.cursor_id });
      } else if (msg?.cursor_position) {
        this.emit("cursor_position", msg?.cursor_position);
      } else if (msg?.misc) {
        if (!this.handleMisc(msg?.misc)) break;
      } else if (msg?.audio_frame) {
        globals.playAudio(msg?.audio_frame.data);
      }
    }
  }

  msgbox(type_: string, title: string, text: string) {
    this._msgbox?.(type_, title, text);
  }

  draw(frame: any) {
    this._draw?.(frame);
  }

  close() {
    this._closed = true;
    this.authenticated = false;
    this.emit("closed", {});
    this._msgs = [];
    clearInterval(this._interval);
    this._ws?.close();
    this._videoDecoder?.close();
  }

  refresh() {
    const misc = message.Misc.fromPartial({ refresh_video: true });
    this._ws?.sendMessage({ misc });
  }

  setMsgbox(callback: MsgboxCallback) {
    this._msgbox = callback;
  }

  setDraw(callback: DrawCallback) {
    this._draw = callback;
  }

  login(password: string | undefined = undefined) {
    if (password) {
      const salt = this._hash?.salt;
      let p = hash([password, salt!]);
      this._password = p;
      const challenge = this._hash?.challenge;
      p = hash([p, challenge!]);
      this.msgbox("connecting", "Connecting...", "Logging in...");
      this._sendLoginMessage(p);
    } else {
      let p = this._password;
      if (p) {
        const challenge = this._hash?.challenge;
        p = hash([p, challenge!]);
      }
      this._sendLoginMessage(p);
    }
  }

  async reconnect() {
    this.close();
    await this.start(this._id);
  }

  _sendLoginMessage(password: Uint8Array | undefined = undefined) {
    const login_request = message.LoginRequest.fromPartial({
      username: this._id!,
      my_id: this._clientID,
      my_name: "RustDesk Web",
      password,
      option: this.fileTransfer ? undefined : this.getOptionMessage(),
      file_transfer: this.fileTransfer ? {dir: "", show_hidden: false} : undefined,
      video_ack_required: !this.fileTransfer,
      version: this.fileTransfer ? "1.4.9" : "",
    });
    this._ws?.sendMessage({ login_request });
  }

  getOptionMessage(): message.OptionMessage | undefined {
    let n = 0;
    const msg = message.OptionMessage.fromPartial({});
    const q = this.getImageQualityEnum(this.getImageQuality(), true);
    const yes = message.OptionMessage_BoolOption.Yes;
    if (q != undefined) {
      msg.image_quality = q;
      n += 1;
    }
    if (this._options["show-remote-cursor"]) {
      msg.show_remote_cursor = yes;
      n += 1;
    }
    if (this._options["lock-after-session-end"]) {
      msg.lock_after_session_end = yes;
      n += 1;
    }
    if (this._options["privacy-mode"]) {
      msg.privacy_mode = yes;
      n += 1;
    }
    if (this._options["disable-audio"]) {
      msg.disable_audio = yes;
      n += 1;
    }
    if (this._options["disable-clipboard"]) {
      msg.disable_clipboard = yes;
      n += 1;
    }
    return n > 0 ? msg : undefined;
  }

  sendVideoReceived() {
    const misc = message.Misc.fromPartial({ video_received: true });
    this._ws?.sendMessage({ misc });
  }

  handleVideoFrame(vf: message.VideoFrame) {
    if (!this._firstFrame) {
      this.msgbox("", "", "");
      this._firstFrame = true;
    }
    if (vf.vp9s) {
      const dec = this._videoDecoder;
      var tm = new Date().getTime();
      var i = 0;
      const n = vf.vp9s?.frames.length;
      vf.vp9s.frames.forEach((f) => {
        dec.processFrame(f.data.slice(0).buffer, (ok: any) => {
          i++;
          if (i == n) this.sendVideoReceived();
          if (ok && dec.frameBuffer && n == i) {
            this.draw(dec.frameBuffer);
            const now = new Date().getTime();
            var elapsed = now - tm;
            this._videoTestSpeed[1] += elapsed;
            this._videoTestSpeed[0] += 1;
            if (this._videoTestSpeed[0] >= 30) {
              console.log(
                "video decoder: " +
                  parseInt(
                    "" + this._videoTestSpeed[1] / this._videoTestSpeed[0]
                  )
              );
              this._videoTestSpeed = [0, 0];
            }
          }
        });
      });
    }
  }

  handlePeerInfo(pi: message.PeerInfo) {
    this._peerInfo = pi;
    this.authenticated = true;
    if (this.fileTransfer) {
      this.msgbox("success", "Connected", "File transfer connected");
      this.emit("peer_info", pi);
      return;
    }
    if (pi.displays.length == 0) {
      this.msgbox("error", "Remote Error", "No Display");
      return;
    }
    this.msgbox("success", "Successful", "Connected, waiting for image...");
    this.emit("peer_info", pi);
    const p = this.shouldAutoLogin();
    if (p) this.inputOsPassword(p);
    const previousInfo = this.getOption("info");
    const username = previousInfo?.username;
    if (username && !pi.username) pi.username = username;
    this._options["last_connected"] = new Date().getTime();
    this.setOption("info", {
      ...previousInfo,
      ...pi,
      alias: previousInfo?.alias || "",
      tags: previousInfo?.tags || [],
      online: previousInfo?.online || false,
      last_online_time: previousInfo?.last_online_time || 0,
    });
    if (this.getRemember()) {
      if (this._password?.length) {
        const p = this._password.toString();
        if (p != this.getOption("password")) {
          this.setOption("password", p);
          console.log("remember password of " + this._id);
        }
      }
    } else {
      this.setOption("password", undefined);
    }
  }

  shouldAutoLogin(): string {
    const l = this.getOption("lock-after-session-end");
    const a = !!this.getOption("auto-login");
    const p = this.getOption("os-password");
    if (p && l && a) {
      return p;
    }
    return "";
  }

  handleMisc(misc: message.Misc) {
    if (misc.audio_format) {
      globals.initAudio(
        misc.audio_format.channels,
        misc.audio_format.sample_rate
      );
    } else if (misc.chat_message) {
      this.emit("chat", { text: misc.chat_message.text });
    } else if (misc.permission_info) {
      const p = misc.permission_info;
      console.info("Change permission " + p.permission + " -> " + p.enabled);
      let name;
      switch (p.permission) {
        case message.PermissionInfo_Permission.Keyboard:
          name = "keyboard";
          break;
        case message.PermissionInfo_Permission.Clipboard:
          name = "clipboard";
          break;
        case message.PermissionInfo_Permission.File:
          name = "file";
          break;
        case message.PermissionInfo_Permission.BlockInput:
          name = "blockInput";
          break;
        case message.PermissionInfo_Permission.Audio:
          name = "audio";
          break;
        default:
          return true;
      }
      if (name in this.permissions) this.permissions[name as keyof typeof this.permissions] = p.enabled;
      this.emit("permission", { [name]: p.enabled });
    } else if (misc.back_notification) {
      this.emit("back_notification", misc.back_notification);
    } else if (misc.switch_display) {
      this.loadVideoDecoder();
      this.emit("switch_display", misc.switch_display);
    } else if (misc.close_reason) {
      this.msgbox("error", "Connection Error", misc.close_reason);
      this.close();
      return false;
    }
    return true;
  }

  getRemember(): Boolean {
    return this._options["remember"] || false;
  }

  applyWebClientDefaults() {
    this._options["view-style"] = "shrink";
    this._options["show-remote-cursor"] = true;
    this._options["disable-audio"] = true;
    this._options["disable-clipboard"] = false;
    this._options["remember"] = false;
  }

  setRemember(v: Boolean) {
    this.setOption("remember", v);
  }

  getOption(name: string): any {
    return this._options[name];
  }

  setOption(name: string, value: any) {
    if (value == undefined) {
      delete this._options[name];
    } else {
      this._options[name] = value;
    }
    this._options["tm"] = new Date().getTime();
    const peers = globals.getPeers();
    peers[this._id] = this._options;
    // Session peer options stay in memory; passwords never persist to browser storage.
  }

  inputKey(
    name: string,
    down: boolean,
    press: boolean,
    alt: Boolean,
    ctrl: Boolean,
    shift: Boolean,
    command: Boolean
  ) {
    if (!this.authenticated || !this.permissions.keyboard) return;
    const key_event = mapKey(name, globals.isDesktop());
    if (!key_event) return;
    if (alt && (name == "VK_MENU" || name == "RAlt")) {
      alt = false;
    }
    if (ctrl && (name == "VK_CONTROL" || name == "RControl")) {
      ctrl = false;
    }
    if (shift && (name == "VK_SHIFT" || name == "RShift")) {
      shift = false;
    }
    if (command && (name == "Meta" || name == "RWin")) {
      command = false;
    }
    key_event.down = down;
    key_event.press = press;
    key_event.modifiers = this.getMod(alt, ctrl, shift, command);
    this._ws?.sendMessage({ key_event });
  }

  ctrlAltDel() {
    if (!this.authenticated || !this.permissions.keyboard) return;
    const key_event = message.KeyEvent.fromPartial({ down: true });
    if (this._peerInfo?.platform == "Windows") {
      key_event.control_key = message.ControlKey.CtrlAltDel;
    } else {
      key_event.control_key = message.ControlKey.Delete;
      key_event.modifiers = this.getMod(true, true, false, false);
    }
    this._ws?.sendMessage({ key_event });
  }

  inputString(seq: string) {
    if (!this.authenticated || !this.permissions.keyboard) return;
    const key_event = message.KeyEvent.fromPartial({ seq });
    this._ws?.sendMessage({ key_event });
  }

  switchDisplay(display: number) {
    const switch_display = message.SwitchDisplay.fromPartial({ display });
    const misc = message.Misc.fromPartial({ switch_display });
    this._ws?.sendMessage({ misc });
  }

  async inputOsPassword(seq: string) {
    this.inputMouse();
    await sleep(50);
    this.inputMouse(0, 3, 3);
    await sleep(50);
    this.inputMouse(1 | (1 << 3));
    this.inputMouse(2 | (1 << 3));
    await sleep(1200);
    const key_event = message.KeyEvent.fromPartial({ press: true, seq });
    this._ws?.sendMessage({ key_event });
  }

  mobileAction(action: 'back'|'home'|'apps'|'volume-up'|'volume-down'|'power') {
    if (!this.authenticated || !this.permissions.keyboard || this._peerInfo?.platform !== 'Android') return;
    if (action === 'back') { this.inputMouse(1 | (4 << 3)); this.inputMouse(2 | (4 << 3)); return; }
    if (action === 'home') { this.inputMouse(1 | (2 << 3)); this.inputMouse(2 | (2 << 3)); return; }
    if (action === 'apps') { this.inputMouse(1 | (2 << 3)); setTimeout(() => this.inputMouse(2 | (2 << 3)), 500); return; }
    const key = action === 'volume-up' ? 77 : action === 'volume-down' ? 78 : 79;
    const down = message.KeyEvent.fromPartial({down: true, control_key: key as message.ControlKey});
    const up = message.KeyEvent.fromPartial({control_key: key as message.ControlKey});
    this._ws?.sendMessage({key_event: down}); setTimeout(() => { if (this.authenticated) this._ws?.sendMessage({key_event: up}); }, 100);
  }

  lockScreen() {
    if (!this.authenticated || !this.permissions.keyboard) return;
    const key_event = message.KeyEvent.fromPartial({
      down: true,
      control_key: message.ControlKey.LockScreen,
    });
    this._ws?.sendMessage({ key_event });
  }

  getMod(alt: Boolean, ctrl: Boolean, shift: Boolean, command: Boolean) {
    const mod: message.ControlKey[] = [];
    if (alt) mod.push(message.ControlKey.Alt);
    if (ctrl) mod.push(message.ControlKey.Control);
    if (shift) mod.push(message.ControlKey.Shift);
    if (command) mod.push(message.ControlKey.Meta);
    return mod;
  }

  inputMouse(
    mask: number = 0,
    x: number = 0,
    y: number = 0,
    alt: Boolean = false,
    ctrl: Boolean = false,
    shift: Boolean = false,
    command: Boolean = false
  ) {
    if (!this.authenticated || !this.permissions.keyboard) return;
    const mouse_event = message.MouseEvent.fromPartial({
      mask,
      x,
      y,
      modifiers: this.getMod(alt, ctrl, shift, command),
    });
    this._ws?.sendMessage({ mouse_event });
  }

  toggleOption(name: string) {
    if (!this.authenticated) return;
    if (["block-input", "unblock-input"].includes(name) && (!this.permissions.keyboard || !this.permissions.blockInput || this._peerInfo?.platform !== "Windows")) return;
    if (name === "lock-after-session-end" && (!this.permissions.keyboard || this._peerInfo?.platform === "Android")) return;
    const v = !this._options[name];
    const option = message.OptionMessage.fromPartial({});
    const v2 = v
      ? message.OptionMessage_BoolOption.Yes
      : message.OptionMessage_BoolOption.No;
    switch (name) {
      case "show-remote-cursor":
        option.show_remote_cursor = v2;
        break;
      case "disable-audio":
        option.disable_audio = v2;
        break;
      case "disable-clipboard":
        option.disable_clipboard = v2;
        break;
      case "lock-after-session-end":
        option.lock_after_session_end = v2;
        break;
      case "privacy-mode":
        option.privacy_mode = v2;
        break;
      case "block-input":
        option.block_input = message.OptionMessage_BoolOption.Yes;
        break;
      case "unblock-input":
        option.block_input = message.OptionMessage_BoolOption.No;
        break;
      default:
        return;
    }
    if (name.indexOf("block-input") < 0) this.setOption(name, v);
    const misc = message.Misc.fromPartial({ option });
    this._ws?.sendMessage({ misc });
  }

  sendClipboard(text: string) {
    if (!this.authenticated || !this.permissions.clipboard || new TextEncoder().encode(text).length > 1024 * 1024) throw new Error("Clipboard unavailable or larger than 1 MiB");
    this._ws?.sendMessage({clipboard: {content: new TextEncoder().encode(text)}});
  }

  getImageQuality() {
    return this.getOption("image-quality");
  }

  getImageQualityEnum(
    value: string,
    ignoreDefault: Boolean
  ): message.ImageQuality | undefined {
    switch (value) {
      case "low":
        return message.ImageQuality.Low;
      case "best":
        return message.ImageQuality.Best;
      case "balanced":
        return ignoreDefault ? undefined : message.ImageQuality.Balanced;
      default:
        return undefined;
    }
  }

  setImageQuality(value: string) {
    this.setOption("image-quality", value);
    const image_quality = this.getImageQualityEnum(value, false);
    if (image_quality == undefined) return;
    const option = message.OptionMessage.fromPartial({ image_quality });
    const misc = message.Misc.fromPartial({ option });
    this._ws?.sendMessage({ misc });
  }

  async loadVideoDecoder() {
    this._videoDecoder?.close();
    this._videoDecoder = await new Promise((resolve) => loadVp9(resolve));
  }
}

function hash(datas: (string | Uint8Array)[]): Uint8Array {
  const hasher = new sha256.Hash();
  datas.forEach((data) => {
    if (typeof data == "string") {
      data = new TextEncoder().encode(data);
    }
    return hasher.update(data);
  });
  return hasher.digest();
}
