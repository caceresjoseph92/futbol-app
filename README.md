# ⚽ Futbol App

Aplicación web para organizar partidos de fútbol 6: divide equipos balanceados automáticamente, gestiona jugadores con calificaciones confidenciales, y guarda el historial de partidos.

## Stack

- **Backend:** Go 1.22 + Chi router
- **Frontend:** HTMX + Tailwind CSS (sin framework JS)
- **Base de datos:** PostgreSQL (Neon — free tier)
- **Auth:** JWT en cookie HttpOnly
- **Deploy:** Render (free tier)

## Arquitectura

Clean Architecture (Ports & Adapters) dentro de un monolito modular.
Ver documentación completa en [docs/architecture.md](docs/architecture.md).

## Inicio rápido

### 1. Clonar y configurar

```bash
cp .env.example .env
# Editar .env con tu DATABASE_URL y JWT_SECRET
```

### 2. Instalar dependencias

```bash
make tidy
```

### 3. Crear la base de datos

Crear un proyecto gratuito en [neon.tech](https://neon.tech) y copiar la connection string en `.env`.

### 4. Aplicar migraciones

```bash
make migrate-up
```

### 5. Ejecutar

```bash
make run
# Servidor en http://localhost:8080
```

## Tests

```bash
make test           # tests unitarios
make test-coverage  # con reporte de cobertura
```

## Comandos disponibles

```bash
make help
```

## Documentación

- [Arquitectura y patrones de diseño](docs/architecture.md)
- [Casos de uso detallados](docs/use-cases.md)
