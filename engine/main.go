package main

import (
	"flag"
	"fmt"
	"hexago/tools/cli"
	"hexago/tools/creator"
	"hexago/tools/runner"
	"os"
)

func main() {
	//принимаем аргументы
	argsAll := os.Args[1:]
	if len(argsAll) == 0 {
		fmt.Fprintln(os.Stderr, "Требуются аргументы")
		os.Exit(1)
	}

	command := cli.Args(argsAll[0])
	//парсим существующие флаги
	fs := flag.NewFlagSet("create", flag.ExitOnError)

	packageFlag := fs.String(string(cli.PackageFlag), string(cli.PackageFlagDefault), "")
	osFlag := fs.String(string(cli.OsFlag), string(cli.OsFlagDefault), "")

	fs.Parse(argsAll[1:])
	args := fs.Args()
	
	switch command {
		case cli.Create:
			{
				if len(args) < 1 {
					fmt.Fprintf(os.Stderr, "Аргумент %v требует дополнительных опций ", cli.Create)
					os.Exit(1)
				}
				folder := string(args[0])
				meta := creator.CreatorMeta{
					InitDir: folder,
					Flags:   map[string]string{string(cli.PackageFlag): *packageFlag, string(cli.OsFlag): *osFlag},
				}
				if err := creator.New(meta); err != nil {
					fmt.Fprintf(os.Stderr, "hexago create: %v\n", err)
					os.Exit(1)
				}

			}
		case cli.Run:
			{
				var platform cli.Platform
				if len(args) < 1 {
					platform = cli.Anfroid
				}else{
					platform = cli.Platform(args[0])
				}
				
				
				runner.Run(platform)

			}
		default:
			{
				fmt.Fprintf(os.Stderr, "Неизвестная команда: %v\n", command)
				os.Exit(1)
			}
	}

}
