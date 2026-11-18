// Действия с виджетами: лампочка (Ctrl+.) и пункт контекстного меню «Действия с виджетом…».
//
// Правки считает hexago assist по синтаксическому дереву Go; здесь — только меню
// и применение правки. Позиции hexago считает в байтах UTF-8, а VS Code — в символах UTF-16,
// поэтому их приходится переводить.

import * as vscode from 'vscode';
import { exec, findHexago, log } from './sdk';

interface Assist {
  id: string;
  title: string;
}

interface Edit {
  start: number;
  end: number;
  text: string;
  /** Обычная правка, которую применить раньше основной (импорт пакета). Смещения основной — уже после неё. */
  before?: Edit;
}

/** Байтовая позиция (UTF-8) для позиции в документе. */
function byteOffset(doc: vscode.TextDocument, pos: vscode.Position): number {
  return Buffer.byteLength(doc.getText().slice(0, doc.offsetAt(pos)), 'utf8');
}

/** Позиция в документе для байтовой позиции (UTF-8). */
function positionAt(doc: vscode.TextDocument, byte: number): vscode.Position {
  const chars = Buffer.from(doc.getText(), 'utf8').subarray(0, byte).toString('utf8').length;
  return doc.positionAt(chars);
}

/** Файл приложения Hexago: подключает пакет виджетов. */
function usesWidgets(doc: vscode.TextDocument): boolean {
  return doc.languageId === 'go' && doc.getText().includes('"hexago/ui/widgets"');
}

async function runAssist(doc: vscode.TextDocument, args: string[]): Promise<string | undefined> {
  const hexago = await findHexago().catch(() => undefined);
  if (!hexago) {
    return undefined;
  }
  return exec(hexago.cmd, ['assist', ...args, '-file', doc.fileName, '-stdin'], { env: hexago.env }, doc.getText());
}

/** Список действий для виджета в позиции pos. */
async function listAssists(doc: vscode.TextDocument, pos: vscode.Position): Promise<Assist[]> {
  if (!usesWidgets(doc)) {
    return [];
  }
  try {
    const out = await runAssist(doc, ['list', '-offset', String(byteOffset(doc, pos))]);
    return out ? (JSON.parse(out) as Assist[]) : [];
  } catch (e) {
    log(`hexago assist list: ${(e as Error).message}`);
    return [];
  }
}

/** Применяет действие id к виджету в позиции pos. */
export async function applyAssist(uri: vscode.Uri, pos: vscode.Position, id: string): Promise<void> {
  const doc = await vscode.workspace.openTextDocument(uri);
  let edit: Edit;
  try {
    const out = await runAssist(doc, ['apply', '-offset', String(byteOffset(doc, pos)), '-id', id]);
    if (!out) {
      return;
    }
    edit = JSON.parse(out) as Edit;
  } catch (e) {
    vscode.window.showErrorMessage((e as Error).message);
    return;
  }
  const editor = await vscode.window.showTextDocument(doc);
  const before = edit.before;
  if (before) {
    // Например, импорт "hexago/ui/geom" для Padding{Padding: geom.All(8)}.
    const r = new vscode.Range(positionAt(doc, before.start), positionAt(doc, before.end));
    if (!await editor.edit((b) => b.replace(r, before.text))) {
      return;
    }
  }
  const range = new vscode.Range(positionAt(doc, edit.start), positionAt(doc, edit.end));
  // Сниппет: курсор встанет в место для ввода — например, на имя виджета в «Обернуть в виджет…».
  await editor.insertSnippet(new vscode.SnippetString(edit.text), range);
}

/** Лампочка и Ctrl+. в Go-файлах с виджетами. */
export class WidgetActionProvider implements vscode.CodeActionProvider {
  static readonly kinds = [vscode.CodeActionKind.RefactorRewrite];

  async provideCodeActions(doc: vscode.TextDocument, range: vscode.Range): Promise<vscode.CodeAction[]> {
    const assists = await listAssists(doc, range.start);
    return assists.map((a) => {
      const action = new vscode.CodeAction(a.title, vscode.CodeActionKind.RefactorRewrite);
      action.command = { command: 'hexago.applyAssist', title: a.title, arguments: [doc.uri, range.start, a.id] };
      return action;
    });
  }
}

/** Пункт контекстного меню «Действия с виджетом…»: список действий для виджета под курсором. */
export async function pickWidgetAction(): Promise<void> {
  const editor = vscode.window.activeTextEditor;
  if (!editor) {
    return;
  }
  const pos = editor.selection.active;
  const assists = await listAssists(editor.document, pos);
  if (!assists.length) {
    vscode.window.showInformationMessage('Под курсором нет виджета Hexago.');
    return;
  }
  const picked = await vscode.window.showQuickPick(
    assists.map((a) => ({ label: a.title, id: a.id })),
    { placeHolder: 'Что сделать с виджетом?' });
  if (picked) {
    await applyAssist(editor.document.uri, pos, picked.id);
  }
}
