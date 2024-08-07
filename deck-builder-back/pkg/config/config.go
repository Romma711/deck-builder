package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port       string
	DBHost     string
	DBUser     string
	DBPassword string
}

func LoadConfig() *Config{
	err := godotenv.Load()
	if err != nil{
		log.Fatalf(("Error cargando el archivo .env: %v"),err)
	}

	config := &Config{
		Port:       getEnv("PORT","8080"),
		DBHost:     getEnv("DB_HOST","localhost"),
		DBUser:     getEnv("DB_HOST","root"),
		DBPassword: getEnv("DB_PASSWORD",""),
	}
	return config
}

func getEnv(key, defaultValue string) string{
	value, exists := os.LookupEnv(key)
	if !exists{
		return defaultValue
	}
	return value
}