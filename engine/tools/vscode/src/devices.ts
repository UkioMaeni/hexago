// Выбор устройства, как у Flutter: справа в строке состояния видно, где запустится приложение,
// по щелчку — подключённые телефоны, запущенные эмуляторы и выключенные эмуляторы (запустить).
//
// Список устройств ведёт hexago devices --json --watch: он печатает строку JSON при каждом изменении.

import * as cp from 'child_process';
import * as readline from 'readline';
import * as vscode from 'vscode';
import { exec, Hexago, log } from './sdk';

/** Устройство из hexago devices --json. */
export interface Device {
  id: string;        // серийный номер — его принимает hexago run -d
  name: string;
  state: string;     // device, booting, unauthorized, offline...
  emulator: boolean;
  avd?: string;
  android?: string;
  abi?: string;
  supported: boolean;
}

/** Эмулятор из hexago emulators --json. */
interface Emulator {
  name: string;      // имя AVD — его принимают --launch и hexago run -d
  label: string;
  running?: string;
}

/** Что выбрали в списке: устройство или выключенный эмулятор. */
type Choice = { device: Device } | { emulator: Emulator };

const selectedKey = 'hexago.device';
const launchTimeout = 3 * 60 * 1000;

export class Devices implements vscode.Disposable {
  private readonly item: vscode.StatusBarItem;
  private hexago?: Hexago;
  private child?: cp.ChildProcess;
  private devices: Device[] = [];
  private watching = false;
  private error?: string;
  private launching?: string; // AVD, который запускается
  private launchTimer?: NodeJS.Timeout;
  private disposed = false;

  constructor(private readonly state: vscode.Memento) {
    this.item = vscode.window.createStatusBarItem('hexago.device', vscode.StatusBarAlignment.Right, 100);
    this.item.name = 'Hexago: устройство';
    this.item.command = 'hexago.selectDevice';
    this.render();
    this.item.show();
  }

  /** Начинает следить за устройствами. */
  start(hexago: Hexago): void {
    this.hexago = hexago;
    this.watch();
  }

  dispose(): void {
    this.disposed = true;
    this.child?.kill();
    clearTimeout(this.launchTimer);
    this.item.dispose();
  }

  /** Устройство, на котором запустится приложение: выбранное, иначе первое готовое. */
  current(): Device | undefined {
    const id = this.state.get<string>(selectedKey);
    return this.devices.find((d) => d.id === id)
      ?? this.devices.find((d) => d.state === 'device')
      ?? this.devices[0];
  }

  /**
   * Куда запускать — значение для hexago run -d. Если устройств нет, предлагает выбрать
   * или запустить эмулятор; выключенный эмулятор hexago run запустит сам и дождётся его.
   */
  async runTarget(): Promise<string | undefined> {
    const d = this.current();
    if (d) {
      return d.id;
    }
    if (this.launching) {
      return this.launching;
    }
    const choice = await this.pick('На чём запустить приложение?');
    if (!choice) {
      return undefined;
    }
    if ('device' in choice) {
      await this.select(choice.device.id);
      return choice.device.id;
    }
    this.setLaunching(choice.emulator.name);
    return choice.emulator.name;
  }

  /** Щелчок по строке состояния: выбрать устройство или запустить эмулятор. */
  async choose(): Promise<void> {
    if (!this.child) {
      this.watch(); // наблюдение упало — пробуем снова
    }
    const choice = await this.pick('Выберите устройство');
    if (!choice) {
      return;
    }
    if ('device' in choice) {
      await this.select(choice.device.id);
      return;
    }
    await this.launch(choice.emulator);
  }

  private async select(id: string): Promise<void> {
    await this.state.update(selectedKey, id);
    this.render();
  }

  private async launch(e: Emulator): Promise<void> {
    if (!this.hexago) {
      return;
    }
    this.setLaunching(e.name);
    try {
      await exec(this.hexago.cmd, ['emulators', '--launch', e.name], { env: this.hexago.env });
    } catch (err) {
      this.setLaunching(undefined);
      vscode.window.showErrorMessage((err as Error).message);
    }
  }

  private setLaunching(avd: string | undefined): void {
    clearTimeout(this.launchTimer);
    this.launching = avd;
    if (avd) {
      this.launchTimer = setTimeout(() => this.setLaunching(undefined), launchTimeout);
    }
    this.render();
  }

  private watch(): void {
    if (this.child || !this.hexago || this.disposed) {
      return;
    }
    const started = Date.now();
    const child = cp.spawn(this.hexago.cmd, ['devices', '--json', '--watch'], { env: this.hexago.env });
    this.child = child;
    this.watching = true;
    let stderr = '';
    readline.createInterface({ input: child.stdout }).on('line', (line) => {
      try {
        this.update(JSON.parse(line) as Device[]);
      } catch {
        log(`hexago devices: ${line}`);
      }
    });
    child.stderr.on('data', (chunk: Buffer) => {
      const text = chunk.toString();
      log(`hexago devices: ${text.trim()}`);
      stderr = (stderr + text).slice(-2000);
    });
    child.on('error', (err) => {
      stderr += err.message;
    });
    child.on('close', (code) => {
      this.child = undefined;
      this.watching = false;
      if (this.disposed) {
        return;
      }
      this.devices = [];
      this.error = stderr.trim() || `hexago devices завершился с кодом ${code}`;
      this.render();
      // Проработал долго — случайный сбой, перезапускаем. Упал сразу (нет Android SDK) —
      // ждём, пока пользователь щёлкнет по строке состояния.
      if (Date.now() - started > 10_000) {
        this.watch();
      }
    });
  }

  private update(list: Device[]): void {
    this.error = undefined;
    this.devices = list;
    if (this.launching) {
      const d = list.find((x) => x.avd === this.launching);
      if (d) {
        void this.state.update(selectedKey, d.id);
        if (d.state === 'device') {
          this.setLaunching(undefined);
        }
      }
    }
    this.render();
  }

  private render(): void {
    const d = this.current();
    const item = this.item;
    item.backgroundColor = undefined;
    if (this.launching && (!d || d.avd !== this.launching || d.state !== 'device')) {
      const name = d?.avd === this.launching ? d.name : this.launching.replace(/_/g, ' ');
      item.text = `$(sync~spin) ${name} — запуск…`;
      item.tooltip = 'Эмулятор запускается. Холодный старт занимает до пары минут.';
    } else if (this.error) {
      item.text = '$(warning) Нет устройств';
      item.tooltip = `Не удалось получить список устройств:\n${this.error}\n\nЩёлкните, чтобы попробовать снова.`;
    } else if (!d) {
      item.text = this.watching ? '$(device-mobile) Нет устройства' : '$(device-mobile) Устройство…';
      item.tooltip = 'Щёлкните, чтобы запустить эмулятор. Телефон подключите по USB с включённой отладкой.';
    } else {
      item.text = `${stateIcon(d)} ${d.name} (${d.emulator ? 'эмулятор' : 'телефон'})`;
      item.tooltip = `${describe(d)}\n\nЗдесь запустится приложение. Щёлкните, чтобы выбрать другое устройство.`;
      if (d.state === 'unauthorized' || !d.supported) {
        item.backgroundColor = new vscode.ThemeColor('statusBarItem.warningBackground');
      }
    }
  }

  /** Показывает список устройств и эмуляторов. */
  private pick(title: string): Promise<Choice | undefined> {
    type Item = vscode.QuickPickItem & { choice?: Choice };
    const qp = vscode.window.createQuickPick<Item>();
    qp.title = title;
    qp.placeholder = 'Устройство для запуска приложения';
    qp.matchOnDescription = true;

    const current = this.current();
    const deviceItems: Item[] = this.devices.map((d) => ({
      label: `${stateIcon(d)} ${d.name}`,
      description: [d.id, d.id === current?.id ? 'выбрано' : ''].filter(Boolean).join(' · '),
      detail: describe(d),
      choice: { device: d },
    }));
    let emulatorsError: string | undefined;
    const problem = (title: string, err: string): Item => ({
      label: `$(warning) ${title}`,
      detail: err.split(/\r?\n/).filter(Boolean).slice(0, 2).join(' '),
    });
    const build = (emulators: Emulator[] | undefined): Item[] => {
      const items: Item[] = [];
      if (this.error) {
        items.push(problem('Не удалось получить список устройств', this.error));
      }
      if (emulatorsError) {
        items.push(problem('Не удалось получить список эмуляторов', emulatorsError));
      }
      if (deviceItems.length) {
        items.push({ label: 'Устройства', kind: vscode.QuickPickItemKind.Separator }, ...deviceItems);
      }
      const off = (emulators ?? []).filter((e) => !e.running && e.name !== this.launching);
      if (off.length) {
        items.push({ label: 'Эмуляторы', kind: vscode.QuickPickItemKind.Separator });
        for (const e of off) {
          items.push({ label: `$(play) Запустить ${e.label}`, description: e.name, choice: { emulator: e } });
        }
      }
      if (emulators && !items.length) {
        items.push({
          label: '$(info) Нет устройств и эмуляторов',
          detail: 'Создайте эмулятор в Android Studio (Device Manager) или подключите телефон по USB с включённой отладкой.',
        });
      }
      return items;
    };
    qp.items = build(undefined);
    qp.busy = true;
    if (this.hexago) {
      exec(this.hexago.cmd, ['emulators', '--json'], { env: this.hexago.env })
        .then((out) => JSON.parse(out) as Emulator[], (err: Error) => {
          log(`hexago emulators: ${err.message}`);
          emulatorsError = err.message;
          return [] as Emulator[];
        })
        .then((emulators) => {
          qp.items = build(emulators);
          qp.busy = false;
        });
    }

    return new Promise((resolve) => {
      let result: Choice | undefined;
      qp.onDidAccept(() => {
        result = qp.selectedItems[0]?.choice;
        if (result) {
          qp.hide();
        }
      });
      qp.onDidHide(() => {
        qp.dispose();
        resolve(result);
      });
      qp.show();
    });
  }
}

function stateIcon(d: Device): string {
  switch (d.state) {
    case 'device':
      return '$(device-mobile)';
    case 'booting':
      return '$(sync~spin)';
    default:
      return '$(warning)';
  }
}

function describe(d: Device): string {
  const parts = [d.emulator ? 'Эмулятор' : 'Телефон'];
  if (d.android) {
    parts.push(`Android ${d.android}`);
  }
  if (d.abi) {
    parts.push(d.abi);
  }
  switch (d.state) {
    case 'device':
      break;
    case 'booting':
      parts.push('загружается');
      break;
    case 'unauthorized':
      parts.push('отладка не разрешена — подтвердите запрос на экране телефона');
      break;
    default:
      parts.push(d.state);
  }
  if (!d.supported) {
    parts.push(`архитектура ${d.abi} не поддерживается`);
  }
  return parts.join(', ');
}
