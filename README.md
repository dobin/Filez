# Filez

Minimal Go webapp for uploading, downloading, and managing files in a directory tree. Single binary, embedded UI, token-cookie auth.

100% vibe coded, GLM 5.2. 


## Run

```powershell
# Build
& "C:\Program Files\Go\bin\go.exe" build -o filez.exe .

# Or: Cross-compile a Linux binary (amd64) from Windows
# (Will polute $env)
$env:GOOS="linux"; $env:GOARCH="amd64"; & "C:\Program Files\Go\bin\go.exe" build -o filez .

# Start (root defaults to .\data, addr to :8080)
.\filez.exe --root .\data --token secret
# or:  $env:FILEZ_TOKEN="secret"; .\filez.exe
```

Open `http://localhost:8080/`, sign in with the token.

### Flags
| Flag      | Default   | Description                          |
|-----------|-----------|--------------------------------------|
| `--root`  | `./data`  | Root directory to serve (created if missing) |
| `--addr`  | `:8080`   | Listen address                       |
| `--token` | —         | Shared auth token (or `FILEZ_TOKEN` env) |

## Features
- List files/folders, navigate into subfolders (breadcrumb + `..` row)
- Upload via button or drag-and-drop into the current folder
- Download (browser-native) or download as JS Blob object
- Delete files and folders (recursive, permanent)
- Create new folders
- Path-escape protection (rejects `..` traversal outside root)

## PowerShell: upload & download

```powershell
# --- Authenticate (save cookie session) ---
$session = $null
Invoke-WebRequest -Uri "http://localhost:8080/login" -Method Post `
    -Body @{ token = "secret" } -SessionVariable session `
    -MaximumRedirection 0 -UseBasicParsing | Out-Null

# --- Upload a file into root (path="") or a subfolder (path="/sub") ---
$boundary = "----fileztest"
$body = @"
--$boundary`r`nContent-Disposition: form-data; name=`"files[]`"; filename=`"hello.txt`"`r`nContent-Type: text/plain`r`n`r`nhello world`r`n--$boundary--`r`n
"@
Invoke-WebRequest -Uri "http://localhost:8080/api/upload?path=" -Method Post `
    -ContentType "multipart/form-data; boundary=$boundary" -Body $body `
    -WebSession $session -UseBasicParsing

# --- Download a file ---
Invoke-WebRequest -Uri "http://localhost:8080/api/download?path=/hello.txt" `
    -WebSession $session -UseBasicParsing -OutFile hello.txt

# --- Delete a file or folder ---
Invoke-WebRequest -Uri "http://localhost:8080/api/delete?path=/hello.txt" `
    -Method Post -WebSession $session -UseBasicParsing

# --- Create a folder ---
Invoke-WebRequest -Uri "http://localhost:8080/api/mkdir?path=&name=sub" `
    -Method Post -WebSession $session -UseBasicParsing
```

## API

| Method | Endpoint                  | Purpose                          |
|--------|---------------------------|----------------------------------|
| GET    | `/`                       | Index page (`?path=/sub`)        |
| GET    | `/login`                  | Login form                       |
| POST   | `/login`                  | Submit token                     |
| POST   | `/logout`                 | Clear cookie                     |
| POST   | `/api/upload?path=`       | Multipart `files[]` upload       |
| GET    | `/api/download?path=`     | Browser download                 |
| GET    | `/api/blob?path=`         | Raw bytes (for JS Blob fetch)    |
| POST   | `/api/delete?path=`       | Delete file/folder (recursive)   |
| POST   | `/api/mkdir?path=&name=`  | Create folder                    |

All `/api/*` and `/` routes require the `filez_token` cookie.

## Notes
- Single dependency: `github.com/go-chi/chi/v5`.
- Templates and static assets are embedded (`//go:embed`) — binary is self-contained.
- Cookie is HttpOnly, SameSite=Lax, 30-day MaxAge. Token stored verbatim; use HTTPS/reverse proxy if exposed beyond localhost.
