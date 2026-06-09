package config

import "os"

type Config struct{Addr string; DB string; UploadDir string}

func Load() Config{
 return Config{Addr:env("API_ADDR",":8080"),DB:env("DATABASE_URL",""),UploadDir:env("UPLOAD_DIR","data/uploads")}
}

func env(k,d string)string{v:=os.Getenv(k);if v==""{return d};return v}
