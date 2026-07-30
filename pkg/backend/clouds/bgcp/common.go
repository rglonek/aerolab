package bgcp

import (
	"embed"
	"strings"
)

//go:embed scripts/*
var scripts embed.FS

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// splits the URL using '/' and returns the last element
func getValueFromURL(url string) string {
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}

func zoneToRegion(zone string) string {
	parts := strings.Split(zone, "-")
	if len(parts) < 2 {
		return zone
	}
	return parts[0] + "-" + parts[1]
}
