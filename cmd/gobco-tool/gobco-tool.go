package main

import (
	"flag"
	"fmt"
	"github.com/junhwi/gobco/instrument"
	"log"
	"io"
	"os"
	"path/filepath"
	"strings"
	"os/exec"
	"path"
	"bufio"
)

// insertAfterPackageMain 在文件中的 'package main' 行后插入一行文本
func insertAfterPackageMain(filePath, lineText string) error {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 读取所有行到一个切片
	scanner := bufio.NewScanner(file)
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// 查找 'package main' 所在的行
	insertIndex := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "package main" {
			insertIndex = i + 1
			break
		}
	}

	if insertIndex == -1 {
		return fmt.Errorf("'package main' not found in the file")
	}

	// 在 'package main' 行后插入新行
	lines = append(lines[:insertIndex+1], lines[insertIndex:]...)
	lines[insertIndex] = lineText

	// 重新打开文件以写入
	file, err = os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 写入修改后的内容
	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	// 确保所有内容都写入到文件
	return writer.Flush()
}
// copyFile 复制文件内容从源路径到目标路径
func copyFile(sourcePath, destinationPath string) error {
	// 打开源文件
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// 创建目标文件
	destinationFile, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	// 使用 io.Copy 复制内容
	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}

	// 确保数据被写入目标文件
	err = destinationFile.Sync()
	if err != nil {
		return err
	}

	return nil
}

func getFd(out, fileName string, i int) (*os.File, error) {
	if out == "" {
		if strings.Contains(fileName, "main") {
			worker := os.Getenv("WORK")
			copyFile(filepath.Join(worker, fileName), filepath.Join(worker,"_testmain.cover.go"))
			insertAfterPackageMain(filepath.Join(worker,"_testmain.cover.go"), `import _ "runtime/coverage"`)
			return  os.Open("/dev/null")
		}
	
		
		worker := os.Getenv("WORK")
		fileNameShort := filepath.Base(fileName)
		extension := filepath.Ext(fileNameShort)
		fileNameWithoutExt := fileNameShort[:len(fileNameShort)-len(extension)]
		if extension==".go" {
			//if i==0 {
				return os.Create(filepath.Join(worker, fileNameWithoutExt+".cover.go"))
			//} else {
			//	copyFile(filepath.Join(worker, fileName), filepath.Join(worker, fileNameWithoutExt+".cover.go"))
			//	return os.Stdout, nil
			//}
		}

		return os.Stdout, nil
	} else {
		return os.Create(out)
	}
}

func runGobco() {

	cmd := flag.NewFlagSet("gobco", flag.ExitOnError)
	// Register all flags same as go tool cover
	outPtr := cmd.String("o", "", "file for output; default: stdout")
	version := cmd.String("V", "", "print version and exit")
	cmd.String("mode", "", "coverage mode: set, count, atomic")
	cmd.String("pkgcfg", "", "coverage pkgcfg")
	cmd.String("outfilelist", "", "coverage outfilelist")
	coverVar := cmd.String("var", "Cov", "name of coverage variable to generate (default \"Cov\")")
	cmd.Parse(os.Args[2:])
	files := cmd.Args()

	if *version != "" {
		fmt.Println("cover version go1.13.1")
	} else {
		for i , file := range files {
			fd, err := getFd(*outPtr, file, i)
			//if i==0 {
				err = instrument.Instrument(file, fd, fmt.Sprintf("%s_%d", *coverVar, i), i==0)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
				}
			//}
		}
	}
}

func fixArgs(args []string) []string {
	var newArgs  []string
	for _, arg := range args  {
		if !strings.HasPrefix(arg ,"-coveragecfg=") {
			newArgs = append(newArgs, arg)
		}
	}
	return newArgs
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("gobco: ")

	tool := os.Args[1]
	args := os.Args[2:]

	toolName := path.Base(tool)
	if toolName == "cover" {
		runGobco()
	} else {
		args = fixArgs(args)
		file, _ := os.OpenFile("example.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		fmt.Fprintf(file, "args: %+v\n", args)
		defer file.Close()
		cmd := exec.Command(tool, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
	os.Exit(0)
}
