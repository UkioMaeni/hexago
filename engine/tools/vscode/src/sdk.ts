// Поиск Hexago SDK и команды hexago.
//
// SDK — это папка движка (в ней go.mod модуля hexago). Где её взять:
//   1. настройка hexago.sdkPath — пользовательская или проекта (.vscode/settings.json);
//   2. команда hexago в PATH.
// Если ничего не найдено, расширение спрашивает папку, как это делает расширение Flutter.

import * as cp from 'child_process';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import * as vscode from 'vscode';

/** Как запускать hexago: программа и окружение. */
export interface Hexago {
  cmd: string;
  env: NodeJS.ProcessEnv;
}

const exe = process.platform === 'win32' ? 'hexago.exe' : 'hexago';

let output: vscode.OutputChannel | undefined;
export function log(text: string): void {
  output ??= vscode.window.createOutputChannel('Hexago');
  output.appendLine(text);
}

/** Сообщает, что папка — движок Hexago. Подходит и корень репозитория с папкой engine внутри. */
export function engineDir(dir: string): string | undefined {
  for (const candidate of [dir, path.join(dir, 'engine')]) {
    try {
      const first = fs.readFileSync(path.join(candidate, 'go.mod'), 'utf8').split(/\r?\n/)[0].trim();
      if (first === 'module hexago') {
        return candidate;
      }
    } catch {
      // нет go.mod — не движок
    }
  }
  return undefined;
}

/** Запускает программу и возвращает её вывод. */
export function exec(cmd: string, args: string[], opts: cp.ExecFileOptions = {}, input?: string): Promise<string> {
  return new Promise((resolve, reject) => {
    const child = cp.execFile(cmd, args, { ...opts, maxBuffer: 16 * 1024 * 1024, encoding: 'utf8' },
      (err, stdout, stderr) => {
        if (err) {
          reject(new Error(String(stderr || err.message).trim()));
        } else {
          resolve(String(stdout));
        }
      });
    if (input !== undefined) {
      child.stdin?.end(input);
    }
  });
}

let built: Promise<void> | undefined;
let builtFor = '';

/**
 * Собирает hexago из исходников SDK в <sdk>/bin. Сборка Go инкрементальная: если ничего
 * не менялось, это занимает доли секунды, зато команда всегда соответствует движку.
 */
function build(sdk: string): Promise<void> {
  if (built && builtFor === sdk) {
    return built;
  }
  builtFor = sdk;
  built = exec('go', ['build', '-o', path.join('bin', exe), '.'], { cwd: sdk })
    .then(() => log(`hexago собран: ${path.join(sdk, 'bin', exe)}`))
    .catch((e) => {
      built = undefined;
      throw new Error(`Не удалось собрать hexago из ${sdk}: ${e.message}`);
    });
  return built;
}

/** Находит hexago. Возвращает undefined, если SDK не настроен и hexago нет в PATH. */
export async function findHexago(): Promise<Hexago | undefined> {
  const configured = vscode.workspace.getConfiguration('hexago').get<string>('sdkPath', '');
  if (configured) {
    const sdk = engineDir(configured);
    if (!sdk) {
      return undefined;
    }
    await build(sdk);
    // HEXAGO_ROOT — чтобы hexago точно знал свой движок (см. sdk.EnginePath).
    return { cmd: path.join(sdk, 'bin', exe), env: { ...process.env, HEXAGO_ROOT: sdk } };
  }
  try {
    await exec('hexago', ['version']);
    return { cmd: 'hexago', env: process.env };
  } catch {
    return undefined;
  }
}

/** Как findHexago, но если SDK не найден — предлагает указать его. */
export async function requireHexago(): Promise<Hexago | undefined> {
  const found = await findHexago().catch((e: Error) => {
    vscode.window.showErrorMessage(e.message);
    return undefined;
  });
  if (found) {
    return found;
  }
  const choice = await vscode.window.showWarningMessage(
    'Hexago SDK не найден. Укажите папку движка Hexago (в ней go.mod модуля hexago).',
    'Указать папку');
  if (choice === 'Указать папку' && await selectSdk()) {
    return findHexago();
  }
  return undefined;
}

/**
 * Спрашивает папку SDK, запоминает её — для всех проектов или только для этого —
 * и по желанию добавляет hexago в PATH.
 */
export async function selectSdk(): Promise<boolean> {
  const picked = await vscode.window.showOpenDialog({
    canSelectFolders: true,
    canSelectFiles: false,
    openLabel: 'Это папка Hexago SDK',
    title: 'Папка движка Hexago',
  });
  if (!picked?.length) {
    return false;
  }
  const sdk = engineDir(picked[0].fsPath);
  if (!sdk) {
    vscode.window.showErrorMessage('В этой папке нет движка Hexago: нужен go.mod с «module hexago» (папка engine).');
    return false;
  }

  const scopes: (vscode.QuickPickItem & { target: vscode.ConfigurationTarget })[] = [
    { label: 'Для всех проектов', description: 'настройки пользователя VS Code', target: vscode.ConfigurationTarget.Global },
  ];
  if (vscode.workspace.workspaceFolders?.length) {
    scopes.push({
      label: 'Только для этого проекта',
      description: '.vscode/settings.json в проекте',
      target: vscode.ConfigurationTarget.Workspace,
    });
  }
  const scope = await vscode.window.showQuickPick(scopes, { placeHolder: 'Где запомнить путь к Hexago SDK?' });
  if (!scope) {
    return false;
  }
  await vscode.workspace.getConfiguration('hexago').update('sdkPath', sdk, scope.target);

  try {
    await vscode.window.withProgress(
      { location: vscode.ProgressLocation.Notification, title: 'Сборка hexago…' },
      () => build(sdk));
  } catch (e) {
    vscode.window.showErrorMessage((e as Error).message);
    return false;
  }

  const addToPath = await vscode.window.showInformationMessage(
    `Hexago SDK: ${sdk}. Добавить команду hexago в PATH, чтобы она работала в любом терминале?`,
    'Добавить в PATH', 'Не надо');
  if (addToPath === 'Добавить в PATH') {
    await addDirToPath(path.join(sdk, 'bin'));
  }
  return true;
}

/** Добавляет папку в PATH пользователя (Windows — переменная среды, macOS/Linux — ~/.profile). */
async function addDirToPath(dir: string): Promise<void> {
  try {
    if (process.platform === 'win32') {
      const script =
        `$d = '${dir.replace(/'/g, "''")}'; ` +
        `$p = [Environment]::GetEnvironmentVariable('Path', 'User'); ` +
        `if (-not (($p -split ';') -contains $d)) { ` +
        `[Environment]::SetEnvironmentVariable('Path', (($p.TrimEnd(';')) + ';' + $d).TrimStart(';'), 'User') }`;
      await exec('powershell.exe', ['-NoProfile', '-Command', script]);
    } else {
      const line = `\nexport PATH="$PATH:${dir}"  # Hexago\n`;
      for (const rc of ['.profile', '.zshrc', '.bashrc']) {
        const file = path.join(os.homedir(), rc);
        if (rc === '.profile' || fs.existsSync(file)) {
          const current = fs.existsSync(file) ? fs.readFileSync(file, 'utf8') : '';
          if (!current.includes(dir)) {
            fs.appendFileSync(file, line);
          }
        }
      }
    }
    vscode.window.showInformationMessage('hexago добавлен в PATH. Новые терминалы увидят его после перезапуска VS Code.');
  } catch (e) {
    vscode.window.showErrorMessage(`Не удалось изменить PATH: ${(e as Error).message}`);
  }
}
