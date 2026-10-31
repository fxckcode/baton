package agent

import "os"

func getenvProcess(key string) string {
	return os.Getenv(key)
}
