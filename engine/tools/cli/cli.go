// Package cli — каркас консольной программы hexago: команды, справка, разбор флагов.
//
// Каждая команда (create, run, version, test) — значение типа Command в своём пакете.
// main.go только собирает их в список и вызывает Main.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

// Context — окружение команды. В тестах подменяем папку и потоки вывода,
// поэтому команды не трогают os.Stdout и os.Getwd напрямую.
type Context struct {
	Dir    string    // текущая папка
	Stdout io.Writer // обычный вывод
	Stderr io.Writer // ошибки и предупреждения
}

// Printf печатает в обычный вывод команды.
func (c *Context) Printf(format string, args ...any) {
	fmt.Fprintf(c.Stdout, format, args...)
}

// Command — одна команда hexago.
type Command struct {
	Name    string // как вызывается: hexago <Name>
	Usage   string // строка использования без слова hexago: "create [флаги] <папка>"
	Summary string // одна строка для общего списка команд
	Help    string // подробное описание для hexago help <команда>
	Run     func(ctx *Context, args []string) error
}

// UsageError — команду вызвали неправильно: код выхода 2 и подсказка, как правильно.
type UsageError struct{ Msg string }

func (e *UsageError) Error() string { return e.Msg }

// Usagef создаёт UsageError с текстом по шаблону, как fmt.Sprintf.
func Usagef(format string, args ...any) error {
	return &UsageError{Msg: fmt.Sprintf(format, args...)}
}

// SilentError — ошибка, о которой команда уже сама всё напечатала (например, hexago test
// вывел список проблем). Main ничего не печатает, только возвращает код выхода 1.
var SilentError = errors.New("ошибка уже показана")

// Main выполняет команду и возвращает код выхода: 0 — успех, 1 — ошибка, 2 — неверный вызов.
func Main(args []string, commands []*Command, stdout, stderr io.Writer) int {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, "hexago:", err)
		return 1
	}
	return Run(&Context{Dir: dir, Stdout: stdout, Stderr: stderr}, args, commands)
}

// Run — то же, что Main, но с заданным окружением. Нужен тестам.
func Run(ctx *Context, args []string, commands []*Command) int {
	if len(args) == 0 {
		printHelp(ctx.Stdout, commands)
		return 0
	}

	name, rest := args[0], args[1:]
	if name == "help" || name == "-h" || name == "-help" || name == "--help" {
		if len(rest) == 0 {
			printHelp(ctx.Stdout, commands)
			return 0
		}
		name, rest = rest[0], []string{"-h"}
	}

	cmd := find(commands, name)
	if cmd == nil {
		fmt.Fprintf(ctx.Stderr, "hexago: неизвестная команда %q\nСписок команд: hexago help\n", name)
		return 2
	}

	err := cmd.Run(ctx, rest)
	var usageErr *UsageError
	switch {
	case err == nil:
		return 0
	case errors.Is(err, flag.ErrHelp):
		printCommandHelp(ctx.Stdout, cmd)
		return 0
	case errors.Is(err, SilentError):
		return 1
	case errors.As(err, &usageErr):
		fmt.Fprintf(ctx.Stderr, "hexago %s: %s\nИспользование: hexago %s\n", cmd.Name, usageErr.Msg, cmd.Usage)
		return 2
	default:
		fmt.Fprintf(ctx.Stderr, "hexago %s: %v\n", cmd.Name, err)
		return 1
	}
}

func find(commands []*Command, name string) *Command {
	for _, c := range commands {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func printHelp(w io.Writer, commands []*Command) {
	fmt.Fprintln(w, "Hexago — движок кроссплатформенных приложений на Go.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Использование:")
	fmt.Fprintln(w, "  hexago <команда> [аргументы]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Команды:")
	for _, c := range commands {
		fmt.Fprintf(w, "  %-8s %s\n", c.Name, c.Summary)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Подробнее о команде: hexago help <команда>")
}

func printCommandHelp(w io.Writer, c *Command) {
	fmt.Fprintf(w, "Использование: hexago %s\n\n%s\n", c.Usage, strings.TrimSpace(c.Help))
}

// NewFlagSet создаёт набор флагов команды. Сам он ничего не печатает:
// все сообщения выводит Run — в одном формате для всех команд.
func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	return fs
}

// ParseArgs разбирает флаги в любом месте: и "create --package x app", и "create app --package x".
// Стандартный flag останавливается на первом аргументе-не-флаге, поэтому разбираем по кускам.
// Возвращает аргументы без флагов.
func ParseArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil, err
			}
			return nil, &UsageError{Msg: err.Error()}
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return positional, nil
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
}
