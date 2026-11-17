// Команда hexago — консольная программа движка Hexago.
//
// Сама программа ничего не умеет: она собирает команды из пакетов tools/... и передаёт
// им аргументы. Так же устроен Flutter: скрипт bin/flutter запускает flutter_tools.
//
// Установка (из папки engine):
//
//	go install .
package main

import (
	"os"

	"hexago/tools/assist"
	"hexago/tools/cli"
	"hexago/tools/creator"
	"hexago/tools/doctor"
	"hexago/tools/runner"
)

func main() {
	commands := []*cli.Command{
		creator.Command,
		runner.Command,
		doctor.TestCommand,
		doctor.VersionCommand,
		assist.Command,
	}
	os.Exit(cli.Main(os.Args[1:], commands, os.Stdout, os.Stderr))
}
