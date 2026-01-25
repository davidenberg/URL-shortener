# URL shortener

URL shortener backend built in Go with a PostgreSQL backend and Redis for caching

## Key Features

1. Cache-Aside Pattern for managing data caching between Redis and PostgreSQL
2. Async Analytics via Goroutines to offload analytics tracking from request handling
3. Docker-compose for setting up entire stack

## Tech Stack
- **Backend**: Go (Standard library + pgx PostgreSQL driver and Go-Redis)
- **Database**: PostgreSQL + Redis
- **Orchestration** Docker, Docker Compose

## Quick Start
1. Clone the repository
```bash
git clone https://github.com/davidenberg/URL-shortener.git
```
2. Build and run the stack
```bash
docker compose up --build
```

## API Endpoints

**POST /urls**<br>
Generates a unique 8-character identifier for the URL <br>
*Request body:*
```json
{
  "original_url": "https://www.google.com"
}
```
*Success response:*
```json
{
  "short_url": "<domain>/urls/:identifier"
}
```
**GET /urls/:identifier** <br>
*Success Response:* 302 Found 

**Get /urls/stats/:identifier**<br>
```json
{
  "creation_time": Timestamp,
  "hits": <Number of hits>
}
```
## License

[MIT](https://choosealicense.com/licenses/mit/)