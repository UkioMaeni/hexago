package creator

import "hexago/tools/cli"

// Command — hexago create.
var Command = &cli.Command{
	Name:    "create",
	Usage:   "create [--package com.example.app] <папка>",
	Summary: "создать новое приложение Hexago",
	Help: `
Создаёт папку с приложением: Go-часть (main.go, go.mod, src/, assets/)
и Android-проект android/ — как flutter create.
Папка "." — создать проект в текущей папке. Имя модуля Go берётся из имени папки.

Флаги:
  --package  идентификатор Android-приложения (по умолчанию com.example.newapp)

Примеры:
  hexago create myapp
  hexago create --package dev.mycompany.shop shop
  hexago create .`,
	Run: run,
}

func run(ctx *cli.Context, args []string) error {
	fs := cli.NewFlagSet("create")
	pkg := fs.String("package", "com.example.newapp", "")
	fs.String("os", "", "") // старый флаг: принимается, но пока ни на что не влияет
	positional, err := cli.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return cli.Usagef("укажи одну папку для проекта, например: hexago create myapp")
	}

	if err := New(CreatorMeta{InitDir: positional[0], Package: *pkg}); err != nil {
		return err
	}
	ctx.Printf("Проект создан: %s (пакет %s)\n\n", positional[0], *pkg)
	ctx.Printf("Дальше:\n")
	if positional[0] != "." {
		ctx.Printf("  cd %s\n", positional[0])
	}
	ctx.Printf("  hexago run android   — собрать и запустить на телефоне или эмуляторе\n")
	return nil
}
