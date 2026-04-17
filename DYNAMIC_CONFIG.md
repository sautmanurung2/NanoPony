# Panduan Konfigurasi Dinamis

Panduan ini menjelaskan cara menggunakan fitur konfigurasi dinamis di NanoPony.

## Ringkasan

Fitur konfigurasi dinamis memungkinkan Anda untuk memuat variabel environment tanpa harus mengubah kode framework NanoPony. Ini sangat berguna ketika:

- Menambahkan variabel environment baru tanpa memperbarui framework.
- Mendukung konfigurasi kustom per aplikasi.
- Memuat feature flags atau pengaturan kustom.

## Penggunaan

### 1. Penggunaan Dasar

```go
config := nanopony.NewConfig()

// Memuat semua variabel environment dengan prefix tertentu
config.LoadDynamic("CUSTOM_")

// Mengakses nilai yang telah dimuat
if apiKey, exists := config.Dynamic["CUSTOM_API_KEY"]; exists {
    fmt.Printf("API Key: %s\n", apiKey)
}
```

### 2. Memuat Prefix Spesifik

Ketika Anda memiliki beberapa variabel environment yang terkait, gunakan prefix untuk memuat semuanya:

```go
// Variabel Environment:
// CUSTOM_API_URL=https://api.example.com
// CUSTOM_TIMEOUT=30s
// CUSTOM_RETRY_COUNT=3

config := nanopony.BuildConfig()
config.LoadDynamic("CUSTOM_")

// Ketiga variabel tersebut sekarang dapat diakses:
fmt.Println(config.Dynamic["CUSTOM_API_URL"])    // https://api.example.com
fmt.Println(config.Dynamic["CUSTOM_TIMEOUT"])    // 30s
fmt.Println(config.Dynamic["CUSTOM_RETRY_COUNT"]) // 3
```

### 3. Memuat Semua Variabel Environment

Anda dapat memuat semua variabel environment (gunakan dengan hati-hati):

```go
config := nanopony.BuildConfig()
config.LoadDynamic("") // Prefix kosong memuat segalanya

// Mengakses variabel environment apa pun
fmt.Println(config.Dynamic["HOME"])
fmt.Println(config.Dynamic["PATH"])
```

> [!WARNING]
> Memuat semua variabel environment mungkin menyertakan data sensitif. Gunakan prefix yang spesifik untuk keamanan yang lebih baik.

### 4. Menambahkan Variabel Environment Baru

Saat Anda perlu menambahkan variabel environment baru, Anda tidak perlu mengubah kode NanoPony:

```go
// Cukup tambahkan ke file .env atau environment Anda:
// MY_FEATURE_ENABLED=true
// MY_FEATURE_URL=https://my-feature.example.com
// MY_FEATURE_API_KEY=secret-key

// Kemudian muat variabel tersebut:
config := nanopony.NewConfig()
config.LoadDynamic("MY_FEATURE_")

// Gunakan:
if config.Dynamic["MY_FEATURE_ENABLED"] == "true" {
    // Aktifkan fitur
}
```

## Praktik Terbaik

### 1. Gunakan Prefix

Kelompokkan variabel environment yang terkait dengan prefix yang sama:

```go
// Baik
config.LoadDynamic("AUTH_")      // Memuat AUTH_API_URL, AUTH_SECRET, dll.
config.LoadDynamic("DATABASE_")  // Memuat DATABASE_URL, DATABASE_POOL, dll.

// Hindari
config.LoadDynamic("")  // Memuat segalanya (mungkin menyertakan data sensitif)
```

### 2. Periksa Keberadaan Key

Selalu periksa apakah sebuah key ada sebelum menggunakannya:

```go
if value, exists := config.Dynamic["MY_KEY"]; exists {
    // Gunakan nilai
} else {
    // Menangani konfigurasi yang hilang
}
```

### 3. Gunakan BuildConfig untuk Multiple Konfigurasi

Jika Anda membutuhkan beberapa konfigurasi dengan pemuatan dinamis yang berbeda:

```go
config1 := nanopony.BuildConfig()
config1.LoadDynamic("SERVICE_A_")

config2 := nanopony.BuildConfig()
config2.LoadDynamic("SERVICE_B_")
```

## Contoh: Setup Lengkap

```go
package main

import (
    "fmt"
    "github.com/sautmanurung2/nanopony"
)

func main() {
    // Inisialisasi config
    config := nanopony.NewConfig()
    
    // Memuat variabel environment kustom
    config.LoadDynamic("MY_APP_")
    
    // Menggunakan field config standar
    fmt.Printf("Environment: %s\n", config.App.Env)
    fmt.Printf("Kafka Model: %s\n", config.App.KafkaModels)
    
    // Menggunakan field config dinamis
    if apiUrl, exists := config.Dynamic["MY_APP_API_URL"]; exists {
        fmt.Printf("API URL: %s\n", apiUrl)
    }
    
    if timeout, exists := config.Dynamic["MY_APP_TIMEOUT"]; exists {
        fmt.Printf("Timeout: %s\n", timeout)
    }
}
```

## Panduan Migrasi

Jika saat ini Anda memiliki variabel environment yang di-hardcode dan ingin menjadikannya dinamis:

### Sebelum (Hardcoded)
```go
// Anda harus mengubah kode nanopony untuk menambahkan variabel env baru
type MyConfig struct {
    ExistingField string
}
```

### Sesudah (Dinamis)
```go
// Cukup gunakan map Dynamic
config := nanopony.NewConfig()
config.LoadDynamic("MY_CUSTOM_")

// Akses variabel apa pun tanpa mengubah kode framework
config.Dynamic["MY_CUSTOM_NEW_FIELD"]
```

## Referensi API

### `LoadDynamic(prefix string)`

Memuat variabel environment dengan prefix yang diberikan ke dalam map `Dynamic`.

**Parameter:**
- `prefix` (string): Prefix untuk memfilter variabel environment. Jika kosong, semua variabel environment akan dimuat.

**Contoh:**
```go
config.LoadDynamic("APP_")  // Memuat APP_*, misal: APP_NAME, APP_VERSION
config.LoadDynamic("")      // Memuat semua variabel environment
```

### `Dynamic map[string]string`

Map yang berisi variabel environment yang dimuat secara dinamis.

**Contoh:**
```go
for key, value := range config.Dynamic {
    fmt.Printf("%s = %s\n", key, value)
}
```
