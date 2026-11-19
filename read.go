package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

func ReadConfigJSON(path string) (ReverseProxySettings, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return ReverseProxySettings{}, fmt.Errorf("read %s: %w", path, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()

	var settings ReverseProxySettings
	if err := decoder.Decode(&settings); err != nil {
		return ReverseProxySettings{}, fmt.Errorf("decode %s: %w", path, err)
	}
	if err := ensureSingleJSONValue(decoder); err != nil {
		return ReverseProxySettings{}, fmt.Errorf("decode %s: %w", path, err)
	}

	settings, err = validateSettings(settings)
	if err != nil {
		return ReverseProxySettings{}, err
	}

	return settings, nil
}

func validateSettings(settings ReverseProxySettings) (ReverseProxySettings, error) {
	port, err := strconv.Atoi(strings.TrimSpace(settings.Port))
	if err != nil || port < 1 || port > 65535 {
		return ReverseProxySettings{}, fmt.Errorf("Port must be an integer from 1 through 65535")
	}
	settings.Port = strconv.Itoa(port)
	if settings.RateLimitWindowSeconds == 0 {
		settings.RateLimitWindowSeconds = 60
	}
	if settings.RateLimitWindowSeconds < 1 {
		return ReverseProxySettings{}, fmt.Errorf("RateLimitWindowSeconds must be positive")
	}
	if settings.EnableRateLimiting && settings.RateLimitAfter < 1 {
		return ReverseProxySettings{}, fmt.Errorf("RateLimitAfter must be positive when rate limiting is enabled")
	}
	return settings, nil
}

func ReadRecords(path string) (LocalDNS, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return LocalDNS{}, fmt.Errorf("read %s: %w", path, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()

	var records LocalDNS
	if err := decoder.Decode(&records); err != nil {
		return LocalDNS{}, fmt.Errorf("decode %s: %w", path, err)
	}
	if err := ensureSingleJSONValue(decoder); err != nil {
		return LocalDNS{}, fmt.Errorf("decode %s: %w", path, err)
	}
	if len(records.Records) == 0 {
		return LocalDNS{}, fmt.Errorf("Records must contain at least one route")
	}

	return records, nil
}

func ensureSingleJSONValue(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("more than one JSON value")
		}
		return err
	}
	return nil
}

func rateLimitWindow(settings ReverseProxySettings) time.Duration {
	return time.Duration(settings.RateLimitWindowSeconds) * time.Second
}
