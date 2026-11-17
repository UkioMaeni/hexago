package runner

import (
	"fmt"
	"hexago/tools/cli"
	"os"
	"os/exec"
)

func Run(platform cli.Platform) error{
	if platform == cli.Anfroid {
		runAndroid()
	}else{
		return fmt.Errorf("Неверный аргумент платформы")
	}
	cmd := exec.Command("go", "run", ".")
	cmd.Stdin = os.Stdin   // чтобы программа могла читать ввод
	cmd.Stdout = os.Stdout // её вывод — в твой терминал
	cmd.Stderr = os.Stderr // её ошибки — тоже
	err := cmd.Run()
	
	return  err
}

func runAndroid(){
	fmt.Println("")
}