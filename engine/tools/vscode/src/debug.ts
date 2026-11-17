// Запуск через Run → Start Debugging (F5), как у Flutter: hexago run собирает приложение и
// запускает его на устройстве, лог идёт в Debug Console, ■ останавливает, ⟳ перезапускает.
//
// Точек останова пока нет: Delve не умеет отлаживать Go, собранный в .so и загруженный в чужой
// процесс (на Android приложение — Java-процесс с libhexago.so). Поэтому отладчик здесь — тонкий
// адаптер протокола DAP вокруг hexago run, без внешних библиотек.

import * as cp from 'child_process';
import * as vscode from 'vscode';
import { Devices } from './devices';
import { Hexago, requireHexago } from './sdk';

/** Настройка запуска из launch.json. */
interface LaunchConfig extends vscode.DebugConfiguration {
  device?: string; // серийный номер, имя устройства или эмулятора; пусто — выбранное в строке состояния
  cwd?: string;    // папка проекта
}

export const debugType = 'hexago';

/** Настройка по умолчанию — её же кладёт hexago create в .vscode/launch.json. */
export const defaultConfig: vscode.DebugConfiguration = { name: 'Hexago', type: debugType, request: 'launch' };

/** Дополняет настройку перед запуском: папка проекта и устройство. */
export class ConfigProvider implements vscode.DebugConfigurationProvider {
  constructor(private readonly devices: () => Devices | undefined, private readonly projectDir: () => string | undefined) {}

  provideDebugConfigurations(): vscode.DebugConfiguration[] {
    return [defaultConfig];
  }

  async resolveDebugConfiguration(folder: vscode.WorkspaceFolder | undefined, config: vscode.DebugConfiguration):
    Promise<vscode.DebugConfiguration | undefined> {
    const c = config as LaunchConfig;
    if (!c.type && !c.request && !c.name) {
      // F5 без launch.json — запускаем как настройку по умолчанию.
      Object.assign(c, defaultConfig);
    }
    c.cwd ??= this.projectDir() ?? folder?.uri.fsPath;
    if (!c.cwd) {
      vscode.window.showErrorMessage('Откройте папку проекта Hexago, чтобы запустить приложение.');
      return undefined;
    }
    if (!c.device) {
      const devices = this.devices();
      if (devices) {
        c.device = await devices.runTarget();
        if (!c.device) {
          return undefined; // не выбрали устройство — не запускаем
        }
      }
    }
    return c;
  }
}

/** Создаёт адаптер для каждого запуска. */
export class AdapterFactory implements vscode.DebugAdapterDescriptorFactory {
  async createDebugAdapterDescriptor(session: vscode.DebugSession): Promise<vscode.DebugAdapterDescriptor | undefined> {
    const hexago = await requireHexago();
    if (!hexago) {
      return undefined;
    }
    return new vscode.DebugAdapterInlineImplementation(new RunAdapter(hexago, session.configuration as LaunchConfig));
  }
}

/** Служебные строки hexago run --events (см. runner.EventPrefix). */
const eventPrefix = 'hexago-event: ';

/** Событие hexago run --events. */
interface RunEvent {
  event: 'started' | 'stopped';
  app?: string;
  pid?: string;
  device?: string;
  name?: string;
}

/** Сообщение протокола DAP — нам нужны только эти поля. */
interface Request {
  seq: number;
  type: string;
  command: string;
  arguments?: unknown;
}

/**
 * Адаптер DAP: запускает hexago run android -d <устройство>, передаёт его вывод в Debug Console
 * и останавливает по ■. Перезапуск VS Code делает сам: останавливает и запускает заново.
 */
class RunAdapter implements vscode.DebugAdapter {
  private readonly emitter = new vscode.EventEmitter<vscode.DebugProtocolMessage>();
  readonly onDidSendMessage = this.emitter.event;
  private seq = 1;
  private child?: cp.ChildProcess;
  private stopping = false;
  private app?: RunEvent; // запущенное приложение — его видно на панели Call Stack
  private pending = '';    // начало строки вывода, конец которой ещё не пришёл

  constructor(private readonly hexago: Hexago, private readonly config: LaunchConfig) {}

  handleMessage(message: vscode.DebugProtocolMessage): void {
    const req = message as Request;
    if (req.type !== 'request') {
      return;
    }
    switch (req.command) {
      case 'initialize':
        this.respond(req, { supportsConfigurationDoneRequest: true, supportsTerminateRequest: true });
        this.event('initialized');
        break;
      case 'launch':
        this.launch(req);
        break;
      case 'configurationDone':
        this.respond(req);
        break;
      case 'threads':
        // Один «поток» — само приложение на устройстве, чтобы было видно, что оно работает.
        this.respond(req, { threads: this.app ? [{ id: 1, name: threadName(this.app) }] : [] });
        break;
      case 'pause':
        this.respond(req, undefined, 'Пауза и точки останова в Hexago пока не поддерживаются.');
        break;
      case 'terminate':
      case 'disconnect':
        this.stop();
        this.respond(req);
        break;
      default:
        this.respond(req, undefined, `Hexago пока не умеет: ${req.command}`);
    }
  }

  dispose(): void {
    this.stop();
    this.emitter.dispose();
  }

  private launch(req: Request): void {
    const args = ['run', 'android'];
    if (this.config.device) {
      args.push('-d', this.config.device);
    }
    this.output(`hexago ${args.join(' ')}\n`, 'console');
    const child = cp.spawn(this.hexago.cmd, [...args, '--events'], { cwd: this.config.cwd, env: this.hexago.env });
    this.child = child;
    child.stdout.setEncoding('utf8');
    child.stderr.setEncoding('utf8');
    child.stdout.on('data', (text: string) => this.stdout(text));
    child.stderr.on('data', (text: string) => this.output(text, 'stderr'));
    child.on('error', (err) => this.output(`Не удалось запустить hexago: ${err.message}\n`, 'stderr'));
    child.on('close', (code) => {
      this.stdout('\n'); // дописать последнюю строку, если она без перевода строки
      this.setApp(undefined);
      this.child = undefined;
      if (!this.stopping) {
        this.output(`\nhexago run завершился (код ${code ?? '—'})\n`, 'console');
      }
      this.event('exited', { exitCode: code ?? 0 });
      this.event('terminated');
    });
    this.respond(req);
  }

  /** Вывод hexago run по строкам: служебные строки — события, остальное — в Debug Console. */
  private stdout(text: string): void {
    const lines = (this.pending + text).split('\n');
    this.pending = lines.pop() ?? '';
    let out = '';
    for (const line of lines) {
      if (line.startsWith(eventPrefix)) {
        this.flush(out);
        out = '';
        this.runEvent(line.slice(eventPrefix.length));
      } else {
        out += line + '\n';
      }
    }
    this.flush(out);
  }

  private flush(text: string): void {
    if (text) {
      this.output(text, 'stdout');
    }
  }

  private runEvent(json: string): void {
    try {
      const e = JSON.parse(json) as RunEvent;
      this.setApp(e.event === 'started' ? e : undefined);
    } catch {
      // незнакомая строка — сессию из-за неё не ломаем
    }
  }

  /** Показывает или убирает приложение на панели Call Stack. */
  private setApp(app: RunEvent | undefined): void {
    if (app && !this.app) {
      this.app = app;
      this.event('thread', { reason: 'started', threadId: 1 });
    } else if (!app && this.app) {
      this.app = undefined;
      this.event('thread', { reason: 'exited', threadId: 1 });
    }
  }

  /** Останавливает hexago run вместе с его adb logcat и сборкой. */
  private stop(): void {
    const child = this.child;
    if (!child?.pid || this.stopping) {
      return;
    }
    this.stopping = true;
    if (process.platform === 'win32') {
      // Сигналов в Windows нет: завершаем всё дерево процессов.
      cp.spawn('taskkill', ['/pid', String(child.pid), '/T', '/F'], { windowsHide: true });
    } else {
      child.kill('SIGINT'); // hexago run выходит по Ctrl+C сам и останавливает logcat
    }
  }

  private send(message: Record<string, unknown>): void {
    this.emitter.fire({ ...message, seq: this.seq++ } as vscode.DebugProtocolMessage);
  }

  private respond(req: Request, body?: unknown, error?: string): void {
    this.send({ type: 'response', request_seq: req.seq, command: req.command, success: !error, message: error, body });
  }

  private event(event: string, body?: unknown): void {
    this.send({ type: 'event', event, body });
  }

  private output(text: string, category: 'console' | 'stdout' | 'stderr'): void {
    this.event('output', { category, output: text });
  }
}

/** Как приложение называется на панели Call Stack. */
function threadName(app: RunEvent): string {
  return `${app.app} (pid ${app.pid}) · ${app.name || app.device}`;
}
