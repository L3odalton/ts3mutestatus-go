package config

import (
    "log"
    "os"
)

type Config struct {
    TS3ApiKey   string
    TS3Address  string
    HABaseURL   string
    HAToken     string
    HAEntityID  string
}

func New() *Config {
    return &Config{
        TS3ApiKey:  getEnvOrPanic("TS3_API_KEY"),
        TS3Address: getEnvOrPanic("TS3_ADDRESS"),
        HABaseURL:  getEnvOrPanic("HA_BASE_URL"),
        HAToken:    getEnvOrPanic("HA_TOKEN"),
        HAEntityID: getEnvOrPanic("HA_ENTITY_ID"),
    }
}

func getEnvOrPanic(key string) string {
    value := os.Getenv(key)
    if value == "" {
        log.Fatalf("%s environment variable is not set", key)
    }
    return value
}