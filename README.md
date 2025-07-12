# 🧙‍♂️ Deck Builder API

Una API REST para crear, leer, actualizar y eliminar mazos de juegos de cartas como Magic: The Gathering. Soporta múltiples formatos, validaciones por tipo de mazo, JWT para autenticación y separación por tipo de usuario.

---

## 🚀 Tecnologías

- **Go (Golang)** con **Gin Gonic**
- **MongoDB** con `mongo-driver v2`
- **JWT** para autenticación
- **go-playground/validator** para validaciones
- Documentación con OpenAPI / Swagger

---

## 📦 Endpoints principales

### 🔐 Autenticación

| Método | Ruta | Descripción |
|--------|------|-------------|
| `POST` | `/api/v2/login` | Iniciar sesión y obtener token JWT |
| `POST` | `/api/v2/register` | Registrar nuevo usuario |
| `PUT`  | `/users/{id}` | Actualizar usuario |
| `DELETE` | `/users/{id}` | Eliminar cuenta de usuario |

---

### 🧩 Decks

| Método | Ruta | Descripción |
|--------|------|-------------|
| `POST` | `/decks` | Crear un nuevo deck (`Commander` o `60 cartas`) |
| `PUT` | `/decks/{id}` | Editar un deck existente |
| `DELETE` | `/decks/{id}` | Eliminar un deck |
| `GET` | `/decks/{id}` | Obtener un deck por ID |
| `GET` | `/decks/owner/{owner_id}` | Obtener decks por usuario |
| `GET` | `/decks/format?format=Modern` | Obtener decks por formato |
| `GET` | `/decks/commander?commander=Chainer` | Obtener decks por comandante |
| `GET` | `/decks` | Obtener todos los decks públicos |

---

## 🧪 Ejemplo de creación de deck Commander

```json
POST /decks

{
  "name": "Mono Black Reanimator",
  "description": "Reanimación de criaturas poderosas.",
  "deck_size": 100,
  "format": "Commander",
  "commander": {
    "name": "Chainer, Nightmare Adept",
    "colors": ["Black", "Red"],
    "mana_value": 4,
    "type": "Creature"
  },
  "cards": [
    {
      "card": {
        "name": "Entomb",
        "colors": ["Black"],
        "mana_value": 1,
        "type": "Instant"
      },
      "amount": 1
    }
  ]
}
```

---

## 🧾 Autenticación con JWT

Todas las rutas `POST`, `PUT` y `DELETE` requieren autenticación mediante **Bearer Token**:

```
Authorization: Bearer <jwt_token>
```

Podés obtener el token mediante el endpoint `/api/v2/login`.

---

## 🛠️ Validaciones destacadas

- `commander` es obligatorio si `format` es `Commander`.
- `colors` en cartas debe contener solo valores válidos: `White`, `Blue`, `Black`, `Red`, `Green`, `Colorless`.
- Se valida estructura anidada de cartas y cantidades.
- Tamaño del deck requerido (`60` o `100` según formato).

---

## 📂 Estructura de carpetas recomendada

```
/pkg
  /deck
    - DeckController.go
    - DeckService.go
    - DeckRoutes.go
  /user
    - UserController.go
    - UserService.go
    - Auth.go
  /middleware
    - JwtAuthMiddleware.go
  /config
    - Mongo.go
    - Env.go
  /utils
    - Validator.go
```

---

## 🧪 Requisitos

- Go 1.21+
- MongoDB Atlas o local
- Variables de entorno en `.env`:
  ```env
  MONGODB_URI=mongodb+srv://<user>:<pass>@cluster.mongodb.net
  SECRET_SIGNING_KEY=tu_clave_jwt
  ```

---

## 📚 Documentación interactiva

Podés visualizar toda la documentación de esta API en Swagger o Postman importando el archivo `deck-builder.md`.

---

## 📄 Licencia

MIT © [Romma711]
