package actorsdk

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"
)

func SanitizeStrings(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		cleaned = append(cleaned, value)
	}
	return cleaned
}

func ClampInt(value, minValue, maxValue, fallback int) int {
	if value == 0 {
		value = fallback
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func RandomID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(time.Now().UnixNano(), 10)))
	}
	return hex.EncodeToString(buf[:])
}

func LogJSON(message string, payload map[string]any) {
	if payload == nil {
		log.Println(message)
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("%s %v", message, payload)
		return
	}
	log.Printf("%s %s", message, string(data))
}

func Coalesce(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func Truncate(value []byte, limit int) string {
	if len(value) <= limit {
		return string(value)
	}
	return string(value[:limit]) + "..."
}
