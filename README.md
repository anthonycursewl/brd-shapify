![Shapify Logo](1776277095_16ffd545277f8026.png)

# Shapify - Image Processing API

A high-performance REST API for image processing with built-in authentication, rate limiting, and caching.

![Go](https://img.shields.io/badge/Go-1.26-blue)
![Fiber](https://img.shields.io/badge/Fiber-v2.52-blue)
![MongoDB](https://img.shields.io/badge/MongoDB-ready-green)
![Redis](https://img.shields.io/badge/Redis-cached-green)

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Environment Variables](#environment-variables)
- [API Endpoints](#api-endpoints)
  - [Auth](#auth)
  - [API Keys](#api-keys)
  - [Image Processing](#image-processing)
- [Request/Response Examples](#requestresponse-examples)
- [Pagination](#pagination)
- [Rate Limiting](#rate-limiting)
- [Error Handling](#error-handling)
- [Project Structure](#project-structure)
- [Tech Stack](#tech-stack)
- [Contributing](#contributing)
- [License](#license)

---

## Overview

Shapify is a production-ready image processing API built with Go (Golang) using the Fiber framework. It provides endpoints for user authentication, API key management, and image operations including resize, convert, and compress.

## Features

- **User Authentication**: JWT-based login and registration
- **API Key Management**: Create, list, and revoke API keys
- **Image Processing**:
  - Resize images to custom dimensions
  - Convert between formats (JPEG, PNG, WebP)
  - Compress with quality control
- **Rate Limiting**: Configurable requests per minute
- **Caching**: Redis-based API key validation cache
- **Pagination**: List endpoints with page/limit controls
- **Structured Logging**: Colored logs for development

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Client                               │
└─────────────────────┬─────────────────────────────────────┘
                      │
              ┌───────▼───────┐
              │  Fiber API    │  (:8080)
              └───────┬───────┘
                      │
        ┌──────────────┼──────────────┐
        │             │             │
    ┌───▼────┐  ┌──▼────┐  ┌──▼────┐
    │ Auth   │  │API    │  │ Image│
    │Routes │  │ Keys  │  │ Process│
    └───┬────┘  └───┬────┘  └──┬──┘
        │            │           │
    ┌───▼──────────▼─────▼────▼────────┐
    │      Middleware Stack           │
    │  • KeyAuth (X-API-Key)        │
    │  • RateLimiter              │
    └────────────┬───────────────┘
                 │
    ┌────────────┼────────────────┐
    │            │                │
┌───▼───┐  ┌────▼────┐  ┌─────▼──────┐
│MongoDB│  │ Redis   │  │ Local Disk │
│(keys)│  │(cache) │  │ (uploads) │
└──────┘  └────────┘  └───────────┘
```

---

## Prerequisites

- **Go** 1.26 or later
- **MongoDB** 4.4+ (for user and API key storage)
- **Redis** 6.0+ (optional, for caching)

---

## Quick Start

### 1. Clone and Install Dependencies

```bash
git clone <repository-url>
cd brd-shapify
go mod download
```

### 2. Configure Environment

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

### 3. Run the Server

```bash
go run ./cmd/api/main.go
```

The server will start on `http://localhost:8080`

### 4. Run Tests

```bash
go run ./cmd/scripts/test_db.go
```

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MONGO_URI` | Yes | - | MongoDB connection string |
| `MONGO_DB` | No | `shapify` | Database name |
| `REDIS_HOST` | No | - | Redis host address |
| `REDIS_PORT` | No | `6379` | Redis port |
| `REDIS_PASSWORD` | No | - | Redis password |
| `REDIS_DB` | No | `0` | Redis database number |
| `PORT` | No | `8080` | Server port |
| `JWT_SECRET` | No | `brd-shapify-secret-key-2024!` | JWT signing secret |
| `API_KEYS` | No | - | Comma-separated fallback API keys |
| `API_KEY` | No | - | Single fallback API key |

### Example `.env` File

```env
MONGO_URI=mongodb://username:password@host:27017/db_name
MONGO_DB=shapify
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_PASSWORD=your_password
REDIS_DB=0
PORT=8080
JWT_SECRET=your-secret-key
```

---

## API Endpoints

### Auth

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/auth/register` | No | Register new user |
| POST | `/auth/login` | No | Login user |

### API Keys (JWT Required)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/api/keys` | JWT | Create new API key |
| GET | `/api/keys` | JWT | List user's API keys |
| DELETE | `/api/keys/:id` | JWT | Delete API key |
| POST | `/api/keys/batch-delete` | JWT | Delete multiple keys |

### Images (X-API-Key + Rate Limit)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/v1/images/resize` | X-API-Key | Resize/compress image |
| POST | `/v1/images/convert` | X-API-Key | Convert image format |
| GET | `/v1/images/:id` | X-API-Key | Get processed image |

### User Images (JWT Required)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/api/images` | JWT | List user's processed images |

---

## Request/Response Examples

### Register User

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john",
    "email": "john@example.com",
    "password": "securepassword"
  }'
```

**Response:**
```json
{
  "success": true,
  "user": {
    "id": "...",
    "username": "john",
    "email": "john@example.com",
    "role": "user",
    "active": true
  }
}
```

### Login

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "securepassword"
  }'
```

**Response:**
```json
{
  "success": true,
  "token": "eyJhbGc...",
  "user": {...}
}
```

### Create API Key

```bash
curl -X POST http://localhost:8080/api/keys \
  -H "Authorization: Bearer <JWT_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My API Key",
    "role": "user",
    "rate_limit": 10
  }'
```

**Response:**
```json
{
  "success": true,
  "key": "sk_abc123...",
  "id": "..."
}
```

### Resize Image (JSON Base64)

```bash
curl -X POST http://localhost:8080/v1/images/resize \
  -H "X-API-Key: sk_abc123..." \
  -H "Content-Type: application/json" \
  -d '{
    "image": "base64image...",
    "width": 256,
    "height": 256,
    "format": "jpg"
  }'
```

**Response:**
```json
{
  "success": true,
  "id": "1234567890_abc123.jpg",
  "original_size": 245000,
  "new_compressed_size": 85000,
  "change_percent": -65.3
}
```

### Resize Image (Binary)

```bash
curl -X POST "http://localhost:8080/v1/images/resize?width=256&height=256&format=jpg" \
  -H "X-API-Key: sk_abc123..." \
  -H "Content-Type: image/jpeg" \
  --data-binary @image.jpg
```

### Compress Image

```bash
curl -X POST http://localhost:8080/v1/images/resize \
  -H "X-API-Key: sk_abc123..." \
  -H "Content-Type: application/json" \
  -d '{
    "image": "base64image...",
    "quality": 75,
    "compress": true
  }'
```

### Convert Image Format

```bash
curl -X POST http://localhost:8080/v1/images/convert \
  -H "X-API-Key: sk_abc123..." \
  -H "Content-Type: application/json" \
  -d '{
    "image": "base64image...",
    "format": "png"
  }'
```

### Get Processed Image

```bash
curl -X GET http://localhost:8080/v1/images/1234567890.jpg \
  -H "X-API-Key: sk_abc123..."
```

**Response:** Raw image bytes with appropriate Content-Type

### List API Keys

```bash
curl -X GET "http://localhost:8080/api/keys?page=1&limit=20" \
  -H "Authorization: Bearer <JWT_TOKEN>"
```

**Response:**
```json
{
  "success": true,
  "keys": [...],
  "total": 50,
  "page": 1,
  "limit": 20,
  "totalPages": 3
}
```

### List User Images

```bash
curl -X GET "http://localhost:8080/api/images?page=1&limit=20" \
  -H "Authorization: Bearer <JWT_TOKEN>"
```

**Response:**
```json
{
  "success": true,
  "images": [
    {
      "id": "...",
      "user_id": "...",
      "image_id": "1234567890.jpg",
      "format": "jpg",
      "width": 256,
      "height": 256,
      "original_size": 245000,
      "compressed_size": 85000,
      "change_percent": -65.3,
      "created_at": "2026-04-15T12:00:00Z"
    }
  ],
  "total": 100,
  "page": 1,
  "limit": 20,
  "totalPages": 5
}
```

---

## Pagination

All list endpoints support pagination with `page` and `limit` query parameters:

| Parameter | Default | Min | Max |
|-----------|---------|-----|-----|
| `page` | 1 | 1 | - |
| `limit` | 20 | 1 | 100 |

Invalid values are automatically corrected to defaults.

---

## Rate Limiting

- **Default**: 10 requests per minute per IP
- **Configurable**: Each API key can have its own rate limit (1-100)
- **Rate Limit Header**: Returns `429 Too Many Requests` when exceeded

---

## Error Handling

Errors return appropriate HTTP status codes with JSON responses containing human-readable messages.

### Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden (account disabled) |
| 404 | Not Found |
| 409 | Conflict (duplicate resource) |
| 429 | Too Many Requests |
| 500 | Internal Server Error |
| 503 | Service Unavailable |

### Error Response Examples

**400 - Bad Request:**
```json
{"error": "Invalid request body"}
```
```json
{"error": "Username, email and password are required"}
```
```json
{"error": "image is required"}
```
```json
{"error": "quality must be between 1 and 100"}
```
```json
{"error": "Invalid Content-Type. Use application/json or image/*"}
```

**401 - Unauthorized:**
```json
{"error": "Missing API key"}
```
```json
{"error": "Invalid API key"}
```
```json
{"error": "Invalid email or password"}
```
```json
{"error": "Authorization header required"}
```

**403 - Forbidden:**
```json
{"error": "Account is disabled"}
```

**404 - Not Found:**
```json
{"error": "Image not found"}
```

**409 - Conflict:**
```json
{"error": "Email already registered"}
```

**429 - Too Many Requests:**
```json
{"error": "Rate limit exceeded"}
```

**503 - Service Unavailable:**
```json
{"error": "Service temporarily unavailable, please try again"}
```

---

## Project Structure

```
brd-shapify/
├── cmd/
│   ├── api/
│   │   └── main.go           # API server entry point
│   ├── auth/
│   │   └── main.go          # Auth server (optional)
│   └── scripts/
│       ├── test_db.go      # Database connection test
│       └── cleanup_keys.go # Key cleanup utility
├── internal/
│   ├── adapters/
│   │   ├── handlers/       # HTTP handlers
│   │   ├── imaging/        # Image processing
│   │   └── storage/        # Database adapters
│   ├── config/
│   │   └── config.go      # Configuration
│   ├── core/
│   │   ├── domain/       # Domain models
│   │   ├── middleware/  # Fiber middleware
│   │   ├── ports/      # Interface definitions
│   │   └── services/    # Business logic
│   ├── logger/
│   │   └── logger.go   # Structured logging
│   └── utils/
│       └── utils.go    # Utilities
├── sdk/
│   └── imaging/
│       └── shimify.go # Go client SDK
├── .env
├── .env.example
├── go.mod
├── go.sum
└── go.work
```

---

## Tech Stack

| Technology | Purpose |
|------------|---------|
| [Go 1.26](https://go.dev/) | Language |
| [Fiber](https://gofiber.io/) | HTTP Framework |
| [MongoDB](https://mongodb.com/) | Database |
| [Redis](https://redis.io/) | Cache |
| [JWT](https://github.com/golang-jwt/jwt) | Authentication |
| [Bcrypt](https://github.com/golang.org/x/crypto/bcrypt) | Password Hashing |
| [imaging](https://github.com/disintegration/imaging) | Image Processing |

---

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests
5. Submit a pull request

---

## License

MIT License - see LICENSE file for details.

---

## Support

For issues and questions:
- GitHub Issues: [repository-url]/issues
- Email: support@shapify.dev