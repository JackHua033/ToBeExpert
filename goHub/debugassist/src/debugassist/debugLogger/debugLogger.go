package debugLogger

import (
	"fmt"
	l4g "github.com/alecthomas/log4go"
	"os"
	"path/filepath"
        "time"
)

var (
	LOG_PATH       = "/logstore/debugassist/"
    Log l4g.Logger = make(l4g.Logger)
)

func GetLogLevel() string {
	level := "error"
	if len(os.Getenv("DEBUGGING_LOGLVL")) > 0 {
		level = os.Getenv("DEBUGGING_LOGLVL")
	}
	return level
}

func InitLogging(appname string) error {
	//Check if the logDir exists, if not, create a new one
    logFile := getLogPath(appname)
	logDir := filepath.Dir(logFile)
	err := CreateDir(logDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create the logDir %s!err: %s", logDir, err.Error())
		return fmt.Errorf("Failed to create the logDir %s!err: %s", logDir, err.Error())
	}
	fmt.Printf("create the logDir %s successfully!\n", logDir)

        flw := l4g.NewFileLogWriter(logFile, true)
        flw.SetRotate(true)
        flw.SetRotateSize(10*1024*1024) //10M
        flw.SetRotateMaxBackup(5)
        flw.SetRotateDaily(false)
        Log.AddFilter("file",l4g.DEBUG, flw)
        time.Sleep(100 * time.Microsecond)
        return nil
}

func getLogPath(appname string) string {
	return filepath.Join(LOG_PATH, appname+".log")
}

// Create a directory.
// Since the l4g may be not created now. please keep the fmt.Printf here.
func CreateDir(dirPath string) error {
	d, err := os.Stat(dirPath)
	if err == nil && d.IsDir() {
		return nil
	}

	if err == nil && !d.IsDir() {
		fmt.Fprintf(os.Stderr, "The %s exists but it's a file! \n", dirPath)
		return fmt.Errorf("The %s exists but it's a file! ", dirPath)
	}

	err = os.MkdirAll(dirPath, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create the directory %s! \n", dirPath)
		return fmt.Errorf("Failed to create the directory %s! ", dirPath)
	}

	return nil
}
