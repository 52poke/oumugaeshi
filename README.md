# Oumugaeshi

A reverse proxy that automatically remuxes OGG/OPUS audio files to WebM container format to support playback in Safari.

## Features

- Transparent proxying of audio files stored in S3-compatible storage
- Automatic remuxing of OGG/OPUS files to WebM container format (without transcoding)
- Caching of remuxed files in S3 bucket
- Support for DELETE operations to clean up remuxed files

## Requirements

- Go 1.21 or higher (for building)
- FFmpeg (for remuxing)
- S3-compatible storage service (e.g., MinIO, AWS S3)

## Configuration

Configuration is done via environment variables:

- `S3_BUCKET`: S3 bucket name (default: "mediawiki")
- `S3_ENDPOINT`: S3 endpoint URL (default: "http://localhost:9000")
- `S3_ACCESS_KEY`: S3 access key (default: "minioadmin")
- `S3_SECRET_KEY`: S3 secret key (default: "minioadmin")
- `S3_REGION`: S3 region (default: "us-east-1")
- `LISTEN_ADDR`: Address to listen on (default: ":8080")

## Usage

### Build and run locally

```bash
go build -o oumugaeshi
./oumugaeshi
```

### Using Docker

```bash
docker run -p 8080:8080 \
  -e S3_ENDPOINT=http://minio:9000 \
  -e S3_BUCKET=mediawiki \
  -e S3_ACCESS_KEY=your-access-key \
  -e S3_SECRET_KEY=your-secret-key \
  ghcr.io/52poke/oumugaeshi:latest
```

## How it works

1. When a request for a `.oga.webm` or `.opus.webm` file is received, the proxy checks if the file already exists in the bucket.
2. If the file exists, it's served directly.
3. If not, the proxy:
   - Locates the original `.oga` or `.opus` file
   - Downloads it temporarily
   - Uses FFmpeg to remux it to WebM container format (without transcoding)
   - Uploads the remuxed file to S3
   - Serves the newly remuxed file to the client

## Path Transformation

The proxy transforms paths according to MediaWiki's pattern:

- Request path: `/wiki/transcoded/4/40/abc.oga/abc.oga.webm`
- Original file path: `/wiki/4/40/abc.oga`
