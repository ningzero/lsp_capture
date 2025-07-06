package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type filterReader struct {
	reader io.Reader
	filter func([]byte) []byte
}

func (fr *filterReader) Read(p []byte) (n int, err error) {
	n, err = fr.reader.Read(p)
	if n > 0 {
		filtered := fr.filter(p[:n])
		copy(p, filtered)
		return len(filtered), err
	}
	return n, err
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <cmd> [args...]")
		os.Exit(1)
	}

	cmdName := os.Args[1]
	cmdArgs := os.Args[2:]

	var baseDir string
	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("get Executable fail:", err)
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Println("Getwd fail:", err)
			return
		} else {
			baseDir = cwd
		}
	} else {
		baseDir = filepath.Dir(exePath)
	}

	logFilePath := filepath.Join(baseDir, "lsp_capture.log")
	clientInputFilePath := filepath.Join(baseDir, "client_input.log")
	serverOutputFilePath := filepath.Join(baseDir, "server_output.log")
	serverErrFilePath := filepath.Join(baseDir, "server_err.log")

	f, err := os.OpenFile(logFilePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("open lsp_capture.log fail: %v", err)
	}
	defer f.Close()
	logger := log.New(f, "", log.LstdFlags)

	clientInputFile, err := os.OpenFile(clientInputFilePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Fatalf("can not open client_input.log: %v", err)
	}
	defer clientInputFile.Close()

	serverOutputFile, err := os.OpenFile(serverOutputFilePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Fatalf("can not open server_output.log: %v", err)
	}
	defer serverOutputFile.Close()

	serverErrorFile, err := os.OpenFile(serverErrFilePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Fatalf("can not open server_err.log: %v", err)
	}
	defer serverErrorFile.Close()

	cmd := exec.Command(cmdName, cmdArgs...)

	stdinIn, err := cmd.StdinPipe()
	if err != nil {
		logger.Fatalf("Failed to create stdin pipe: %v", err)
	}
	stdoutOut, err := cmd.StdoutPipe()
	if err != nil {
		logger.Fatalf("Failed to create stdout pipe: %v", err)
	}
	stderrOut, err := cmd.StderrPipe()
	if err != nil {
		logger.Fatalf("Failed to create stderr pipe: %v", err)
	}

	inputWriter := io.MultiWriter(stdinIn, clientInputFile)
	go func() {
		_, err := io.Copy(inputWriter, os.Stdin)
		if err != nil {
			logger.Printf("Error copying stdin: %v", err)
		}
	}()

	filterFunc := func(data []byte) []byte {
		// 过滤掉一些 debug 输出，以免影响 lsp 协议正常运行
		if strings.Contains(string(data), "Listening for transport") {
			return []byte{}
		}
		return data
	}
	filteredStdout := &filterReader{
		reader: stdoutOut,
		filter: filterFunc,
	}

	outputWriter := io.MultiWriter(os.Stdout, serverOutputFile)
	go func() {
		_, err := io.Copy(outputWriter, filteredStdout)
		if err != nil {
			logger.Printf("Error copying stdout: %v", err)
		}
	}()

	errorWriter := io.MultiWriter(os.Stderr, serverErrorFile)
	go func() {
		_, err := io.Copy(errorWriter, stderrOut)
		if err != nil {
			logger.Printf("Error copying stderr: %v", err)
		}
	}()

	if err := cmd.Start(); err != nil {
		logger.Printf("start sub process fail: %v", err)
		os.Exit(1)
	}

	if err := cmd.Wait(); err != nil {
		logger.Fatalf("Command finished with error: %v", err)
	}
}
