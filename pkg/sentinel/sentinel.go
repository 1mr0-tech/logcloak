package sentinel

import (
	"fmt"
	"time"
)

const Prefix = "[LOGCLOAK-DROP]"

func Line(reason, podName string) string {
	return fmt.Sprintf("%s %s | reason=%s | pod=%s | line suppressed",
		Prefix, time.Now().UTC().Format(time.RFC3339), reason, podName)
}
