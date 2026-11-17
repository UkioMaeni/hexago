// Расширение VS Code для Hexago: поиск SDK, команды hexago, выбор устройства и действия с виджетами.

import * as fs from 'fs';
import * as path from 'path';
import * as vscode from 'vscode';
import { applyAssist, pickWidgetAction, WidgetActionProvider } from './assist';
import { Devices } from './devices';
import { exec, findHexago, requireHexago, selectSdk } from './sdk';

/** Что расширение показывает, пока открыт проект Hexago. */
interface Project extends vscode.Disposable {
  devices: Devices;
}

let project: Project | undefined;

export function activate(context: vscode.ExtensionContext): void {
  context.subscriptions.push(
    vscode.commands.registerCommand('hexago.selectSdk', selectSdk),
    vscode.commands.registerCommand('hexago.createProject', createProject),
    vscode.commands.registerCommand('hexago.run', run),
    vscode.commands.registerCommand('hexago.selectDevice', selectDevice),
    vscode.commands.registerCommand('hexago.doctor', () => runTask('Проверка окружения', ['test'])),
    vscode.commands.registerCommand('hexago.widgetActions', pickWidgetAction),
    vscode.commands.registerCommand('hexago.applyAssist', applyAssist),
    vscode.languages.registerCodeActionsProvider(
      { language: 'go', scheme: 'file' },
      new WidgetActionProvider(),
      { providedCodeActionKinds: WidgetActionProvider.kinds }),
  );

  // Проект может появиться и после открытия папки — например, после hexago create . в терминале.
  const refresh = () => refreshProject(context);
  const watcher = vscode.workspace.createFileSystemWatcher('**/go.mod');
  context.subscriptions.push(
    watcher,
    watcher.onDidCreate(refresh),
    watcher.onDidChange(refresh),
    watcher.onDidDelete(refresh),
    vscode.workspace.onDidChangeWorkspaceFolders(refresh),
    {
      dispose: () => {
        project?.dispose();
        project = undefined;
      },
    },
  );
  refresh();
}

export function deactivate(): void {}

/** Показывает кнопку запуска и выбор устройства, когда открыт проект Hexago, и прячет, когда нет. */
function refreshProject(context: vscode.ExtensionContext): void {
  const dir = hexagoProject();
  void vscode.commands.executeCommand('setContext', 'hexago.project', !!dir);
  if (!dir) {
    project?.dispose();
    project = undefined;
    return;
  }
  if (project) {
    return;
  }

  // Кнопка запуска и устройство в строке состояния — как у Flutter.
  const runItem = vscode.window.createStatusBarItem('hexago.run', vscode.StatusBarAlignment.Left, 100);
  runItem.name = 'Hexago: запуск';
  runItem.text = '$(play) Hexago';
  runItem.tooltip = 'Собрать и запустить на выбранном устройстве (hexago run)';
  runItem.command = 'hexago.run';
  runItem.show();
  const devices = new Devices(context.workspaceState);
  const p: Project = {
    devices,
    dispose: () => {
      runItem.dispose();
      devices.dispose();
    },
  };
  project = p;

  // Сразу проверяем, что SDK найден, и предлагаем указать его, если нет.
  findHexago().then(async (found) => found ?? await requireHexago(), (e: Error) => {
    vscode.window.showErrorMessage(e.message);
    return undefined;
  }).then((hexago) => {
    if (hexago && project === p) {
      devices.start(hexago);
    }
  });
}

/** Собирает и запускает приложение на выбранном устройстве. */
async function run(): Promise<void> {
  if (!project) {
    await runTask('Запуск', ['run', 'android']);
    return;
  }
  const target = await project.devices.runTarget();
  if (target) {
    await runTask('Запуск', ['run', 'android', '-d', target]);
  }
}

async function selectDevice(): Promise<void> {
  if (!project) {
    vscode.window.showInformationMessage('Откройте папку проекта Hexago (в его go.mod есть зависимость от hexago).');
    return;
  }
  await project.devices.choose();
}

/** Папка открытого проекта Hexago: в go.mod есть зависимость от hexago. */
function hexagoProject(): string | undefined {
  for (const folder of vscode.workspace.workspaceFolders ?? []) {
    try {
      const mod = fs.readFileSync(path.join(folder.uri.fsPath, 'go.mod'), 'utf8');
      if (/^\s*require\s+hexago\s/m.test(mod) || /^\s*replace\s+hexago\s/m.test(mod)) {
        return folder.uri.fsPath;
      }
    } catch {
      // нет go.mod
    }
  }
  return undefined;
}

/** Запускает hexago как задачу VS Code: вывод в терминале, Ctrl+C останавливает. */
async function runTask(title: string, args: string[]): Promise<void> {
  const hexago = await requireHexago();
  if (!hexago) {
    return;
  }
  const folder = vscode.workspace.workspaceFolders?.[0];
  const env: Record<string, string> = {};
  for (const [k, v] of Object.entries(hexago.env)) {
    if (v !== undefined && (k === 'HEXAGO_ROOT')) {
      env[k] = v;
    }
  }
  const task = new vscode.Task(
    { type: 'hexago', command: args.join(' ') },
    folder ?? vscode.TaskScope.Workspace,
    title,
    'hexago',
    new vscode.ShellExecution(hexago.cmd, args, { cwd: folder?.uri.fsPath, env }),
  );
  task.presentationOptions = { reveal: vscode.TaskRevealKind.Always, clear: true };
  await vscode.tasks.executeTask(task);
}

/** Создаёт проект: папка, имя, пакет Android — затем предлагает открыть его. */
async function createProject(): Promise<void> {
  const hexago = await requireHexago();
  if (!hexago) {
    return;
  }
  const parent = await vscode.window.showOpenDialog({
    canSelectFolders: true, canSelectFiles: false, openLabel: 'Создать проект здесь', title: 'Где создать проект?',
  });
  if (!parent?.length) {
    return;
  }
  const name = await vscode.window.showInputBox({
    title: 'Имя проекта',
    prompt: 'Латинские буквы, цифры, «-» и «_» — станет именем папки и модуля Go',
    value: 'myapp',
    validateInput: (v) => /^[A-Za-z0-9_.-]+$/.test(v) ? undefined : 'Только латинские буквы, цифры, «-», «_» и «.»',
  });
  if (!name) {
    return;
  }
  const pkg = await vscode.window.showInputBox({
    title: 'Идентификатор Android-приложения',
    value: `com.example.${name.toLowerCase().replace(/[^a-z0-9_]/g, '_')}`,
    validateInput: (v) => /^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$/.test(v)
      ? undefined : 'Например com.example.app: минимум две части через точку',
  });
  if (!pkg) {
    return;
  }
  const dir = path.join(parent[0].fsPath, name);
  try {
    await vscode.window.withProgress(
      { location: vscode.ProgressLocation.Notification, title: `Создание проекта ${name}…` },
      () => exec(hexago.cmd, ['create', '--package', pkg, dir], { env: hexago.env }));
  } catch (e) {
    vscode.window.showErrorMessage((e as Error).message);
    return;
  }
  const open = await vscode.window.showInformationMessage(`Проект ${name} создан.`, 'Открыть', 'Открыть в новом окне');
  if (open) {
    await vscode.commands.executeCommand('vscode.openFolder', vscode.Uri.file(dir), open === 'Открыть в новом окне');
  }
}
