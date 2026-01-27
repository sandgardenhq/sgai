package notify

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func sendRemote(title, message string) error {
	ntfyURL := os.Getenv("sgai_NTFY")
	if ntfyURL == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, errRequest := http.NewRequestWithContext(ctx, http.MethodPost, ntfyURL, strings.NewReader(message))
	if errRequest != nil {
		return errRequest
	}
	req.Header.Set("Title", title)

	resp, errDo := http.DefaultClient.Do(req)
	if errDo != nil {
		return errDo
	}
	defer func() {
		errClose := resp.Body.Close()
		if errClose != nil {
			log.Println("close failed:", errClose)
		}
	}()

	return nil
}
