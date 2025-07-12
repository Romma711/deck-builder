---
title: deck-builder
language_tabs:
  - shell: Shell
  - http: HTTP
  - javascript: JavaScript
  - ruby: Ruby
  - python: Python
  - php: PHP
  - java: Java
  - go: Go
toc_footers: []
includes: []
search: true
code_clipboard: true
highlight_theme: darkula
headingLevel: 2
generator: "@tarslib/widdershins v4.0.30"

---

# deck-builder

Base URLs:

# Authentication

- HTTP Authentication, scheme: bearer

# User

## POST Log de usuario

POST /api/v2/login

> Body Parameters

```json
{
  "username": "romma",
  "password": "andres123"
}
```

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|body|body|object| no |none|

> Response Examples

> 201 Response

```json
{
  "id": "string",
  "username": "string",
  "email": "string",
  "password": "string",
  "created_at": "string",
  "updated_at": "string"
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|201|[Created](https://tools.ietf.org/html/rfc7231#section-6.3.2)|none|Inline|

### Responses Data Schema

HTTP Status Code **201**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» id|string|true|none||none|
|» username|string|true|none||none|
|» email|string|true|none||none|
|» password|string|true|none||none|
|» created_at|string|true|none||none|
|» updated_at|string|true|none||none|

## POST Crear usuario

POST /api/v2/register

> Body Parameters

```json
{
  "username": "{{$internet.userName}}",
  "email": "{{$internet.email}}",
  "password": "{{$internet.password}}"
}
```

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|body|body|object| no |none|

> Response Examples

> 200 Response

```json
{}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### Responses Data Schema

## DELETE Eliminar usuario

DELETE /users/6871a67e61dcbd2bfdf3fd01

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|Authorization|header|string| no |none|

> Response Examples

> 204 Response

```
{}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|204|[No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)|none|Inline|

### Responses Data Schema

## PUT Actualizar usuario

PUT /users/6871b69079be0d19f80f05b5

> Body Parameters

```json
{
  "username": "joseman",
  "email": "joseman@gmail.com",
  "password": "123456"
}
```

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|body|body|object| no |none|

> Response Examples

> 200 Response

```json
{
  "id": "string",
  "username": "string",
  "email": "string",
  "password": "string",
  "created_at": "string",
  "updated_at": "string"
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» id|string|true|none||none|
|» username|string|true|none||none|
|» email|string|true|none||none|
|» password|string|true|none||none|
|» created_at|string|true|none||none|
|» updated_at|string|true|none||none|

# Decks

## POST Crear deck commander

POST /decks

> Body Parameters

```json
{
  "name": "Mono Black Reanimator",
  "description": "Reanimación de criaturas poderosas.",
  "deck_size": 100,
  "format": "Commander",
  "commander": {
    "name": "Chainer, Nightmare Adept",
    "colors": [
      "Black",
      "Red"
    ],
    "mana_value": 4,
    "type": "Creature"
  },
  "cards": [
    {
      "card": {
        "name": "Entomb",
        "colors": [
          "Black"
        ],
        "mana_value": 1,
        "type": "Instant"
      },
      "amount": 1
    },
    {
      "card": {
        "name": "Reanimate",
        "colors": [
          "Black"
        ],
        "mana_value": 1,
        "type": "Sorcery"
      },
      "amount": 1
    }
  ]
}
```

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|body|body|object| no |none|

> Response Examples

> 201 Response

```json
{
  "id": "string",
  "owner_id": "string",
  "name": "string",
  "description": "string",
  "deck_size": 0,
  "commander": {
    "name": "string",
    "colors": [
      "string"
    ],
    "mana_value": 0,
    "type": "string"
  },
  "partner": {
    "name": "string",
    "colors": [
      "string"
    ],
    "mana_value": 0,
    "type": "string"
  },
  "format": "string",
  "cards": [
    {
      "card": {
        "name": "string",
        "colors": [
          "string"
        ],
        "mana_value": 0,
        "type": "string"
      },
      "amount": 0
    }
  ],
  "created_at": "string",
  "updated_at": "string"
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|201|[Created](https://tools.ietf.org/html/rfc7231#section-6.3.2)|none|Inline|

### Responses Data Schema

HTTP Status Code **201**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» id|string|true|none||none|
|» owner_id|string|true|none||none|
|» name|string|true|none||none|
|» description|string|true|none||none|
|» deck_size|integer|true|none||none|
|» commander|object|true|none||none|
|»» name|string|true|none||none|
|»» colors|[string]|true|none||none|
|»» mana_value|integer|true|none||none|
|»» type|string|true|none||none|
|» partner|object|true|none||none|
|»» name|string|true|none||none|
|»» colors|[string]|true|none||none|
|»» mana_value|integer|true|none||none|
|»» type|string|true|none||none|
|» format|string|true|none||none|
|» cards|[object]|true|none||none|
|»» card|object|true|none||none|
|»»» name|string|true|none||none|
|»»» colors|[string]|true|none||none|
|»»» mana_value|integer|true|none||none|
|»»» type|string|true|none||none|
|»» amount|integer|true|none||none|
|» created_at|string|true|none||none|
|» updated_at|string|true|none||none|

## DELETE Eliminar deck

DELETE /decks/687292a38e76d410de7ab5ea

> Body Parameters

```json
{}
```

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|body|body|object| no |none|

> Response Examples

> 204 Response

```
{}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|204|[No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)|none|Inline|

### Responses Data Schema

## PUT Actualizar deck commander

PUT /decks/68729796b1d6ed9c7f27e7df

> Body Parameters

```json
{
  "name": "Red and Black Reanimator",
  "description": "Reanimación de criaturas poderosas.",
  "deck_size": 100,
  "format": "Commander",
  "commander": {
    "name": "Chainer, Nightmare Adept",
    "colors": [
      "Black",
      "Red"
    ],
    "mana_value": 4,
    "type": "Creature"
  },
  "cards": [
    {
      "card": {
        "name": "Entomb",
        "colors": [
          "Black"
        ],
        "mana_value": 1,
        "type": "Instant"
      },
      "amount": 1
    },
    {
      "card": {
        "name": "Reanimate",
        "colors": [
          "Black"
        ],
        "mana_value": 1,
        "type": "Sorcery"
      },
      "amount": 1
    }
  ]
}
```

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|body|body|object| no |none|

> Response Examples

> 200 Response

```json
{
  "id": "string",
  "owner_id": "string",
  "name": "string",
  "description": "string",
  "deck_size": 0,
  "commander": {
    "name": "string",
    "colors": [
      "string"
    ],
    "mana_value": 0,
    "type": "string"
  },
  "partner": {
    "name": "string",
    "colors": null,
    "mana_value": 0,
    "type": "string"
  },
  "format": "string",
  "cards": [
    {
      "card": {
        "name": "string",
        "colors": [
          "string"
        ],
        "mana_value": 0,
        "type": "string"
      },
      "amount": 0
    }
  ],
  "created_at": "string",
  "updated_at": "string"
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» id|string|true|none||none|
|» owner_id|string|true|none||none|
|» name|string|true|none||none|
|» description|string|true|none||none|
|» deck_size|integer|true|none||none|
|» commander|object|true|none||none|
|»» name|string|true|none||none|
|»» colors|[string]|true|none||none|
|»» mana_value|integer|true|none||none|
|»» type|string|true|none||none|
|» partner|object|true|none||none|
|»» name|string|true|none||none|
|»» colors|null|true|none||none|
|»» mana_value|integer|true|none||none|
|»» type|string|true|none||none|
|» format|string|true|none||none|
|» cards|[object]|true|none||none|
|»» card|object|true|none||none|
|»»» name|string|true|none||none|
|»»» colors|[string]|true|none||none|
|»»» mana_value|integer|true|none||none|
|»»» type|string|true|none||none|
|»» amount|integer|true|none||none|
|» created_at|string|true|none||none|
|» updated_at|string|true|none||none|

## GET Obtener deck por id

GET /decks/68729796b1d6ed9c7f27e7df

> Response Examples

> 200 Response

```json
{
  "_id": "string",
  "owner_id": "string",
  "name": "string",
  "description": "string",
  "deck_size": 0,
  "commander": {
    "name": "string",
    "colors": [
      "string"
    ],
    "mana_value": 0,
    "type": "string"
  },
  "partner": {
    "name": "string",
    "colors": null,
    "mana_value": 0,
    "type": "string"
  },
  "format": "string",
  "cards": [
    {
      "card": {
        "name": "string",
        "colors": [
          "string"
        ],
        "mana_value": 0,
        "type": "string"
      },
      "amount": 0
    }
  ],
  "created_at": "string",
  "updated_at": "string"
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» _id|string|true|none||none|
|» owner_id|string|true|none||none|
|» name|string|true|none||none|
|» description|string|true|none||none|
|» deck_size|integer|true|none||none|
|» commander|object|true|none||none|
|»» name|string|true|none||none|
|»» colors|[string]|true|none||none|
|»» mana_value|integer|true|none||none|
|»» type|string|true|none||none|
|» partner|object|true|none||none|
|»» name|string|true|none||none|
|»» colors|null|true|none||none|
|»» mana_value|integer|true|none||none|
|»» type|string|true|none||none|
|» format|string|true|none||none|
|» cards|[object]|true|none||none|
|»» card|object|true|none||none|
|»»» name|string|true|none||none|
|»»» colors|[string]|true|none||none|
|»»» mana_value|integer|true|none||none|
|»»» type|string|true|none||none|
|»» amount|integer|true|none||none|
|» created_at|string|true|none||none|
|» updated_at|string|true|none||none|

## PUT Actualizar deck 60 cartas

PUT /decks/6872a662bcb9ebb7d42a7ff5

> Body Parameters

```json
{
  "name": "Standard Mono White Midrange",
  "description": "Control de mesa y criaturas eficientes.",
  "deck_size": 60,
  "format": "Standard",
  "cards": [
    {
      "card": {
        "name": "Elesh Norn, Mother of Machines",
        "colors": [
          "White"
        ],
        "mana_value": 5,
        "type": "Creature"
      },
      "amount": 2
    },
    {
      "card": {
        "name": "Fateful Absence",
        "colors": [
          "White"
        ],
        "mana_value": 2,
        "type": "Instant"
      },
      "amount": 3
    },
    {
      "card": {
        "name": "The Wandering Emperor",
        "colors": [
          "White"
        ],
        "mana_value": 4,
        "type": "Planeswalker"
      },
      "amount": 2
    }
  ]
}
```

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|body|body|object| no |none|

> Response Examples

> 200 Response

```json
{
  "id": "string",
  "owner_id": "string",
  "name": "string",
  "description": "string",
  "deck_size": 0,
  "commander": {
    "name": "string",
    "colors": null,
    "mana_value": 0,
    "type": "string"
  },
  "partner": {
    "name": "string",
    "colors": null,
    "mana_value": 0,
    "type": "string"
  },
  "format": "string",
  "cards": [
    {
      "card": {
        "name": "string",
        "colors": [
          "string"
        ],
        "mana_value": 0,
        "type": "string"
      },
      "amount": 0
    }
  ],
  "created_at": "string",
  "updated_at": "string"
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» id|string|true|none||none|
|» owner_id|string|true|none||none|
|» name|string|true|none||none|
|» description|string|true|none||none|
|» deck_size|integer|true|none||none|
|» commander|object|true|none||none|
|»» name|string|true|none||none|
|»» colors|null|true|none||none|
|»» mana_value|integer|true|none||none|
|»» type|string|true|none||none|
|» partner|object|true|none||none|
|»» name|string|true|none||none|
|»» colors|null|true|none||none|
|»» mana_value|integer|true|none||none|
|»» type|string|true|none||none|
|» format|string|true|none||none|
|» cards|[object]|true|none||none|
|»» card|object|true|none||none|
|»»» name|string|true|none||none|
|»»» colors|[string]|true|none||none|
|»»» mana_value|integer|true|none||none|
|»»» type|string|true|none||none|
|»» amount|integer|true|none||none|
|» created_at|string|true|none||none|
|» updated_at|string|true|none||none|

## GET Obtener decks por dueño

GET /decks/owner/68719dec194b32b316e35d51

> Response Examples

> 200 Response

```json
[
  {
    "_id": "string",
    "owner_id": "string",
    "name": "string",
    "description": "string",
    "deck_size": 0,
    "format": "string",
    "cards": [
      {
        "card": {
          "name": "string",
          "colors": [
            "string"
          ],
          "mana_value": 0,
          "type": "string"
        },
        "amount": 0
      }
    ],
    "created_at": "string",
    "updated_at": "string",
    "commander": {
      "name": "string",
      "colors": [
        "string"
      ],
      "mana_value": 0,
      "type": "string"
    },
    "partner": {
      "name": "string",
      "colors": [
        "string"
      ],
      "mana_value": 0,
      "type": "string"
    }
  }
]
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» _id|string|true|none||none|
|» owner_id|string|true|none||none|
|» name|string|true|none||none|
|» description|string|true|none||none|
|» deck_size|integer|true|none||none|
|» format|string|true|none||none|
|» cards|[object]|true|none||none|
|»» card|object|true|none||none|
|»»» name|string|true|none||none|
|»»» colors|[string]|true|none||none|
|»»» mana_value|integer|true|none||none|
|»»» type|string|true|none||none|
|»» amount|integer|true|none||none|
|» created_at|string|true|none||none|
|» updated_at|string|true|none||none|
|» commander|object|true|none||none|
|»» name|string|true|none||none|
|»» colors|[string]|true|none||none|
|»» mana_value|integer|true|none||none|
|»» type|string|true|none||none|
|» partner|object|true|none||none|
|»» name|string|true|none||none|
|»» colors|[string]¦null|true|none||none|
|»» mana_value|integer|true|none||none|
|»» type|string|true|none||none|

## GET Obtener por formato

GET /decks/format/

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|format|query|string| no |none|

> Response Examples

> 200 Response

```json
[
  {
    "id": "string",
    "owner_id": "string",
    "name": "string",
    "description": "string",
    "deck_size": 0,
    "format": "string",
    "cards": [
      {
        "card": {
          "name": "string",
          "colors": [
            "string"
          ],
          "mana_value": 0,
          "type": "string"
        },
        "amount": 0
      }
    ],
    "created_at": "string",
    "updated_at": "string"
  }
]
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» id|string|false|none||none|
|» owner_id|string|false|none||none|
|» name|string|false|none||none|
|» description|string|false|none||none|
|» deck_size|integer|false|none||none|
|» format|string|false|none||none|
|» cards|[object]|false|none||none|
|»» card|object|true|none||none|
|»»» name|string|true|none||none|
|»»» colors|[string]|true|none||none|
|»»» mana_value|integer|true|none||none|
|»»» type|string|true|none||none|
|»» amount|integer|true|none||none|
|» created_at|string|false|none||none|
|» updated_at|string|false|none||none|

## GET Obtener por commander

GET /decks/commander

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|commander|query|string| no |none|

> Response Examples

> 200 Response

```json
[
  {
    "id": "string",
    "owner_id": "string",
    "name": "string",
    "description": "string",
    "deck_size": 0,
    "commander": {
      "name": "string",
      "colors": [
        "string"
      ],
      "mana_value": 0,
      "type": "string"
    },
    "partner": {
      "name": "string",
      "colors": [
        "string"
      ],
      "mana_value": 0,
      "type": "string"
    },
    "format": "string",
    "cards": [
      {
        "card": {
          "name": "string",
          "colors": [
            "string"
          ],
          "mana_value": 0,
          "type": "string"
        },
        "amount": 0
      }
    ],
    "created_at": "string",
    "updated_at": "string"
  }
]
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» id|string|false|none||none|
|» owner_id|string|false|none||none|
|» name|string|false|none||none|
|» description|string|false|none||none|
|» deck_size|integer|false|none||none|
|» commander|object|false|none||none|
|»» name|string|true|none||none|
|»» colors|[string]|true|none||none|
|»» mana_value|integer|true|none||none|
|»» type|string|true|none||none|
|» partner|object|false|none||none|
|»» name|string|true|none||none|
|»» colors|[string]|true|none||none|
|»» mana_value|integer|true|none||none|
|»» type|string|true|none||none|
|» format|string|false|none||none|
|» cards|[object]|false|none||none|
|»» card|object|true|none||none|
|»»» name|string|true|none||none|
|»»» colors|[string]|true|none||none|
|»»» mana_value|integer|true|none||none|
|»»» type|string|true|none||none|
|»» amount|integer|true|none||none|
|» created_at|string|false|none||none|
|» updated_at|string|false|none||none|

## GET Obtener todos los decks

GET /decks/

> Response Examples

> 200 Response

```json
[
  {
    "id": "string",
    "owner_id": "string",
    "name": "string",
    "deck_size": 0,
    "commander": {
      "name": "string",
      "colors": [
        "string"
      ],
      "mana_value": 0,
      "type": "string"
    },
    "format": "string"
  }
]
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» id|string|true|none||none|
|» owner_id|string|true|none||none|
|» name|string|true|none||none|
|» deck_size|integer|true|none||none|
|» commander|object|true|none||none|
|»» name|string|true|none||none|
|»» colors|[string]¦null|true|none||none|
|»» mana_value|integer|true|none||none|
|»» type|string|true|none||none|
|» format|string|true|none||none|

# Data Schema

