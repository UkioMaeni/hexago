package dev.hexago;

import android.content.ClipData;
import android.content.ClipboardManager;
import android.content.Context;
import android.icu.text.BreakIterator;
import android.text.Editable;
import android.text.InputType;
import android.text.Selection;
import android.text.SpannableStringBuilder;
import android.view.KeyEvent;
import android.view.SurfaceView;
import android.view.inputmethod.BaseInputConnection;
import android.view.inputmethod.EditorInfo;
import android.view.inputmethod.ExtractedText;
import android.view.inputmethod.ExtractedTextRequest;
import android.view.inputmethod.InputConnection;
import android.view.inputmethod.InputMethodManager;

/**
 * Поверхность, на которой рисует Go, и одновременно текстовый редактор для клавиатуры.
 *
 * Поля ввода рисует Go, но клавиатура Android работает только с View-редактором: она получает от него
 * InputConnection и через него вставляет буквы, подсказки и исправления. Поэтому текст поля, к которому
 * подключена клавиатура, живёт здесь (editable): клавиатура меняет его, а мы после каждой правки
 * отправляем его в Go. Когда текст меняет приложение, Go присылает его сюда (update).
 *
 * Позиции выделения здесь — в единицах UTF-16 (как в Java); в руны их переводит Go.
 */
final class HexagoView extends SurfaceView {

    // Как в widgets.KeyboardType и widgets.TextInputAction.
    private static final int KEYBOARD_TEXT = 0, KEYBOARD_NUMBER = 1, KEYBOARD_DECIMAL = 2, KEYBOARD_PHONE = 3,
            KEYBOARD_EMAIL = 4, KEYBOARD_URL = 5;
    private static final int ACTION_DONE = 1, ACTION_GO = 2, ACTION_NEXT = 3, ACTION_SEARCH = 4, ACTION_SEND = 5,
            ACTION_NEWLINE = 6;
    // Флаги поля из Go.
    private static final int FLAG_MULTILINE = 1, FLAG_OBSCURE = 2, FLAG_NO_AUTOCORRECT = 4;

    final SpannableStringBuilder editable = new SpannableStringBuilder();
    private final InputMethodManager imm;
    private boolean active;      // к полю в Go подключена клавиатура
    private int inputType = InputType.TYPE_CLASS_TEXT;
    private int imeOptions = EditorInfo.IME_ACTION_DONE;
    private int action = ACTION_DONE; // что делает Enter однострочного поля
    private boolean multiline;

    HexagoView(Context context) {
        super(context);
        setFocusable(true);
        setFocusableInTouchMode(true);
        imm = (InputMethodManager) context.getSystemService(Context.INPUT_METHOD_SERVICE);
    }

    @Override
    public boolean onCheckIsTextEditor() {
        return active;
    }

    @Override
    public InputConnection onCreateInputConnection(EditorInfo out) {
        if (!active) {
            return null;
        }
        out.inputType = inputType;
        out.imeOptions = imeOptions | EditorInfo.IME_FLAG_NO_FULLSCREEN;
        out.initialSelStart = Selection.getSelectionStart(editable);
        out.initialSelEnd = Selection.getSelectionEnd(editable);
        return new Connection(this);
    }

    /** Go: подключить клавиатуру к полю и показать её. */
    void show(int keyboard, int action, int flags, int capitalize, String text, int selStart, int selEnd,
            int composeStart, int composeEnd) {
        int type = inputTypeOf(keyboard, flags, capitalize);
        int options = imeActionOf(action);
        boolean restart = !active || type != inputType || options != imeOptions || !sameText(text, selStart, selEnd);
        active = true;
        inputType = type;
        imeOptions = options;
        this.action = action;
        multiline = (flags & FLAG_MULTILINE) != 0;
        setEditing(text, selStart, selEnd, composeStart, composeEnd);
        requestFocus();
        if (restart) {
            imm.restartInput(this);
        }
        imm.showSoftInput(this, 0);
    }

    /** Go: спрятать клавиатуру — ни одно поле не в фокусе. */
    void hide() {
        if (!active) {
            return;
        }
        active = false;
        imm.hideSoftInputFromWindow(getWindowToken(), 0);
        imm.restartInput(this);
    }

    /** Go: текст поля поменяло приложение. */
    void update(String text, int selStart, int selEnd, int composeStart, int composeEnd) {
        if (!active) {
            return;
        }
        boolean textChanged = !editable.toString().equals(text);
        setEditing(text, selStart, selEnd, composeStart, composeEnd);
        if (textChanged) {
            imm.restartInput(this); // клавиатура должна забыть своё набираемое слово
        } else {
            reportSelection();
        }
    }

    private boolean sameText(String text, int selStart, int selEnd) {
        return editable.toString().equals(text) && Selection.getSelectionStart(editable) == selStart
                && Selection.getSelectionEnd(editable) == selEnd;
    }

    private void setEditing(String text, int selStart, int selEnd, int composeStart, int composeEnd) {
        if (!editable.toString().equals(text)) {
            editable.replace(0, editable.length(), text);
        }
        BaseInputConnection.removeComposingSpans(editable);
        int n = editable.length();
        Selection.setSelection(editable, clamp(selStart, n), clamp(selEnd, n));
        if (composeEnd > composeStart) {
            BaseInputConnection.setComposingSpans(editable);
            // setComposingSpans помечает весь текст — сузить до нужного куска.
            Object[] spans = editable.getSpans(0, n, Object.class);
            for (Object s : spans) {
                if ((editable.getSpanFlags(s) & android.text.Spanned.SPAN_COMPOSING) != 0) {
                    editable.setSpan(s, clamp(composeStart, n), clamp(composeEnd, n), editable.getSpanFlags(s));
                }
            }
        }
    }

    private static int clamp(int i, int n) {
        return Math.max(0, Math.min(i, n));
    }

    /** Текст изменился: отправить его в Go и сообщить клавиатуре новое выделение. */
    void changed() {
        int selStart = Selection.getSelectionStart(editable), selEnd = Selection.getSelectionEnd(editable);
        int cs = BaseInputConnection.getComposingSpanStart(editable), ce = BaseInputConnection.getComposingSpanEnd(editable);
        HexagoActivity.onEditing(editable.toString(), selStart, selEnd, cs, ce);
        reportSelection();
    }

    private void reportSelection() {
        imm.updateSelection(this, Selection.getSelectionStart(editable), Selection.getSelectionEnd(editable),
                BaseInputConnection.getComposingSpanStart(editable), BaseInputConnection.getComposingSpanEnd(editable));
    }

    /** Кнопка ввода клавиатуры или Enter однострочного поля. */
    void performAction(int imeAction) {
        int a = action;
        switch (imeAction) {
            case EditorInfo.IME_ACTION_GO: a = ACTION_GO; break;
            case EditorInfo.IME_ACTION_NEXT: a = ACTION_NEXT; break;
            case EditorInfo.IME_ACTION_SEARCH: a = ACTION_SEARCH; break;
            case EditorInfo.IME_ACTION_SEND: a = ACTION_SEND; break;
            case EditorInfo.IME_ACTION_DONE: a = ACTION_DONE; break;
        }
        HexagoActivity.onTextAction(a);
    }

    @Override
    public boolean onKeyDown(int keyCode, KeyEvent event) {
        if (active && handleKey(event)) {
            return true;
        }
        return super.onKeyDown(keyCode, event);
    }

    /** Клавиши физической клавиатуры (и те, что экранная присылает как нажатия). */
    boolean handleKey(KeyEvent e) {
        if (e.getAction() != KeyEvent.ACTION_DOWN) {
            return false;
        }
        int start = Selection.getSelectionStart(editable), end = Selection.getSelectionEnd(editable);
        int lo = Math.max(0, Math.min(start, end)), hi = Math.max(start, end);
        boolean shift = e.isShiftPressed();
        // Стирание и стрелки — по целым символам: эмодзи из нескольких кодов (семья, флаг) не разваливается.
        BreakIterator chars = BreakIterator.getCharacterInstance();
        chars.setText(editable.toString());
        switch (e.getKeyCode()) {
            case KeyEvent.KEYCODE_DEL:
                if (lo != hi) {
                    editable.delete(lo, hi);
                } else if (lo > 0) {
                    editable.delete(Math.max(0, chars.preceding(lo)), lo);
                }
                break;
            case KeyEvent.KEYCODE_FORWARD_DEL:
                if (lo != hi) {
                    editable.delete(lo, hi);
                } else if (hi < editable.length()) {
                    int next = chars.following(hi);
                    editable.delete(hi, next == BreakIterator.DONE ? editable.length() : next);
                }
                break;
            case KeyEvent.KEYCODE_DPAD_LEFT:
            case KeyEvent.KEYCODE_DPAD_RIGHT: {
                int dir = e.getKeyCode() == KeyEvent.KEYCODE_DPAD_LEFT ? -1 : 1;
                int to;
                if (!shift && lo != hi) {
                    to = dir < 0 ? lo : hi;
                } else {
                    to = dir < 0 ? (end > 0 ? chars.preceding(end) : 0) : (end < editable.length() ? chars.following(end) : end);
                    if (to == BreakIterator.DONE) {
                        to = dir < 0 ? 0 : editable.length();
                    }
                }
                to = clamp(to, editable.length());
                Selection.setSelection(editable, shift ? start : to, to);
                break;
            }
            case KeyEvent.KEYCODE_MOVE_HOME:
                Selection.setSelection(editable, shift ? start : 0, 0);
                break;
            case KeyEvent.KEYCODE_MOVE_END:
                Selection.setSelection(editable, shift ? start : editable.length(), editable.length());
                break;
            case KeyEvent.KEYCODE_ENTER:
            case KeyEvent.KEYCODE_NUMPAD_ENTER:
                if (!multiline) {
                    HexagoActivity.onTextAction(action);
                    return true;
                }
                editable.replace(lo, hi, "\n");
                break;
            default:
                if (e.isCtrlPressed()) {
                    return handleShortcut(e.getKeyCode(), lo, hi);
                }
                int c = e.getUnicodeChar(e.getMetaState());
                if (c == 0) {
                    return false;
                }
                editable.replace(lo, hi, new String(Character.toChars(c)));
        }
        BaseInputConnection.removeComposingSpans(editable);
        changed();
        return true;
    }

    /** Ctrl+A, Ctrl+C, Ctrl+X, Ctrl+V. */
    private boolean handleShortcut(int keyCode, int lo, int hi) {
        ClipboardManager clip = (ClipboardManager) getContext().getSystemService(Context.CLIPBOARD_SERVICE);
        switch (keyCode) {
            case KeyEvent.KEYCODE_A:
                Selection.setSelection(editable, 0, editable.length());
                break;
            case KeyEvent.KEYCODE_C:
            case KeyEvent.KEYCODE_X:
                if (lo == hi || (inputType & InputType.TYPE_TEXT_VARIATION_PASSWORD) != 0) {
                    return true;
                }
                clip.setPrimaryClip(ClipData.newPlainText("text", editable.subSequence(lo, hi).toString()));
                if (keyCode == KeyEvent.KEYCODE_X) {
                    editable.delete(lo, hi);
                }
                break;
            case KeyEvent.KEYCODE_V:
                ClipData data = clip.getPrimaryClip();
                if (data == null || data.getItemCount() == 0) {
                    return true;
                }
                CharSequence s = data.getItemAt(0).coerceToText(getContext());
                editable.replace(lo, hi, s);
                break;
            default:
                return false;
        }
        BaseInputConnection.removeComposingSpans(editable);
        changed();
        return true;
    }

    private static int inputTypeOf(int keyboard, int flags, int capitalize) {
        int t;
        switch (keyboard) {
            case KEYBOARD_NUMBER: t = InputType.TYPE_CLASS_NUMBER; break;
            case KEYBOARD_DECIMAL: t = InputType.TYPE_CLASS_NUMBER | InputType.TYPE_NUMBER_FLAG_DECIMAL; break;
            case KEYBOARD_PHONE: t = InputType.TYPE_CLASS_PHONE; break;
            case KEYBOARD_EMAIL: t = InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_EMAIL_ADDRESS; break;
            case KEYBOARD_URL: t = InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_URI; break;
            default: t = InputType.TYPE_CLASS_TEXT;
        }
        if ((t & InputType.TYPE_MASK_CLASS) != InputType.TYPE_CLASS_TEXT) {
            return t;
        }
        if ((flags & FLAG_MULTILINE) != 0) {
            t |= InputType.TYPE_TEXT_FLAG_MULTI_LINE;
        }
        if ((flags & FLAG_OBSCURE) != 0) {
            t |= InputType.TYPE_TEXT_VARIATION_PASSWORD;
        } else if ((flags & FLAG_NO_AUTOCORRECT) == 0) {
            t |= InputType.TYPE_TEXT_FLAG_AUTO_CORRECT;
        } else {
            t |= InputType.TYPE_TEXT_FLAG_NO_SUGGESTIONS;
        }
        switch (capitalize) {
            case 1: t |= InputType.TYPE_TEXT_FLAG_CAP_SENTENCES; break;
            case 2: t |= InputType.TYPE_TEXT_FLAG_CAP_WORDS; break;
            case 3: t |= InputType.TYPE_TEXT_FLAG_CAP_CHARACTERS; break;
        }
        return t;
    }

    private static int imeActionOf(int action) {
        switch (action) {
            case ACTION_GO: return EditorInfo.IME_ACTION_GO;
            case ACTION_NEXT: return EditorInfo.IME_ACTION_NEXT;
            case ACTION_SEARCH: return EditorInfo.IME_ACTION_SEARCH;
            case ACTION_SEND: return EditorInfo.IME_ACTION_SEND;
            case ACTION_NEWLINE: return EditorInfo.IME_ACTION_NONE;
            default: return EditorInfo.IME_ACTION_DONE;
        }
    }

    /** Соединение с клавиатурой: она правит editable, а мы после каждой правки отправляем его в Go. */
    private static final class Connection extends BaseInputConnection {
        private final HexagoView view;
        private int batch;

        Connection(HexagoView view) {
            super(view, true);
            this.view = view;
        }

        @Override
        public Editable getEditable() {
            return view.editable;
        }

        private boolean done(boolean result) {
            if (batch == 0) {
                view.changed();
            }
            return result;
        }

        @Override
        public boolean beginBatchEdit() {
            batch++;
            return super.beginBatchEdit();
        }

        @Override
        public boolean endBatchEdit() {
            boolean r = super.endBatchEdit();
            batch = Math.max(0, batch - 1);
            return done(r);
        }

        @Override
        public boolean commitText(CharSequence text, int newCursorPosition) {
            return done(super.commitText(text, newCursorPosition));
        }

        @Override
        public boolean setComposingText(CharSequence text, int newCursorPosition) {
            return done(super.setComposingText(text, newCursorPosition));
        }

        @Override
        public boolean setComposingRegion(int start, int end) {
            return done(super.setComposingRegion(start, end));
        }

        @Override
        public boolean finishComposingText() {
            return done(super.finishComposingText());
        }

        @Override
        public boolean deleteSurroundingText(int before, int after) {
            return done(super.deleteSurroundingText(before, after));
        }

        @Override
        public boolean deleteSurroundingTextInCodePoints(int before, int after) {
            return done(super.deleteSurroundingTextInCodePoints(before, after));
        }

        @Override
        public boolean setSelection(int start, int end) {
            return done(super.setSelection(start, end));
        }

        @Override
        public ExtractedText getExtractedText(ExtractedTextRequest request, int flags) {
            // Без этого клавиатура не видит текст поля целиком и не набирает слова с подсказками.
            Editable e = view.editable;
            ExtractedText et = new ExtractedText();
            et.text = e.toString();
            et.startOffset = 0;
            et.partialStartOffset = -1;
            et.partialEndOffset = -1;
            et.selectionStart = Selection.getSelectionStart(e);
            et.selectionEnd = Selection.getSelectionEnd(e);
            et.flags = view.multiline ? 0 : ExtractedText.FLAG_SINGLE_LINE;
            return et;
        }

        @Override
        public boolean performEditorAction(int actionCode) {
            view.performAction(actionCode);
            return true;
        }

        @Override
        public boolean sendKeyEvent(KeyEvent event) {
            // Клавиатура прислала нажатие (обычно «стереть» или Enter): обработать его здесь,
            // а не отправлять обратно в окно, — иначе оно придёт дважды.
            if (event.getAction() == KeyEvent.ACTION_DOWN) {
                view.handleKey(event);
            }
            return true;
        }
    }
}
