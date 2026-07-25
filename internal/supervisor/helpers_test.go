package supervisor

import "runtime"

func testEchoCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", "echo", "hola"}
	}
	return "echo", []string{"hola"}
}

func testFailingCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", "exit", "1"}
	}
	return "sh", []string{"-c", "exit 1"}
} 

