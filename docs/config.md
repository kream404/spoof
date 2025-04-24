## Config

Please reference the `sample.json` in the repo when consulting the documentation.

### Cache Configuration

The `cache` section of the configuration defines the connection parameters for a database and settings for reproducible data generation.

This section allows the tool to fetch data from an existing database using SQL statements, which can be helpful for integrating real data into your generated output.

---

Here is an example of the `cache` section:

```json
"cache": {
  "hostname": "localhost",
  "port": "5432",
  "username": "user",
  "password": "password",
  "name": "database",
  "seed": "4d810b2c-f1ca-46dc-9240-b7b19f1fc46e",
  "statement": "SELECT customer_id, amount, customer_email FROM account.customer;"
}
```

### Seed

The cache config can optionally be provisioned with a seed. This will be the seed used for all RNG operations in the generation, giving deterministic results.

Any run without a seed will output the seed used in generation to the console, which can be used to replicate outputs.

---

## Field Types

When configuring your CSV generation, each field in the `fields` array represents a column with specific data logic. The name provided will be the name of the column in the output file.

Any field can be seeded from the database using the following syntax. If you want to have the column name in the ouput differ from the column name of the database, you can pass an alias. This will be database column, and the name will be the csv column.

```json
{ ..."seed_type":"db", "alias":"customerid" }
```

---
### Supported field types:
---

### `Override`

This will hardcode a value to the given input. The row will always contain the supplied value.

```json
{ "name": "active", "value": "true" }
```
---

### `iterator`

Sequentially generates an increasing integer starting from 1.

```json
{ "name": "id", "type": "iterator" }
```

> Use this when you need a unique row identifier or simple sequence.

---
### `uuid`

Generates a uuid v7. This is not currently deterministic.

```json
{ "name": "id", "type": "uuid" }
```

---

### `range`

Selects a random value from a defined set of options.

```json
{ "name": "customerstatusid", "type": "range", "values": "1, 2, 3, 4, 5, 6" }
```

> Ideal for enums, status codes, or controlled categories. Can handle both numbers and strings.

---

### `number`

Generates a random floating-point number between a minimum and maximum. An optional format can be passed to specify the number of decimal places

```json
{ "name": "amount", "type": "number", "format": "2", "min": -2000.00, "max": 2000.00 },
```

---

### `timestamp`

Creates a timestamp using the current time formatted with Go-style time syntax. This is not currently deterministic.

```json
{ "name": "updated_at", "type": "timestamp", "format": "02-01-06 15:04:05" }
```

> Supports custom formatting using [Go time layouts](https://pkg.go.dev/time#pkg-constants).

---
### `email`

Generates an email address. This currently just a random string.

```json
{ "name": "customer_email", "type": "email" }
```
---

### `reflection`

Copies the value of another field. Can optionally modifying numeric inputs. The target will be multiplied by the modifier.

```json
{ "name": "inverse", "type": "reflection", "target": "amount", "modifier": -1 }
```

- `target`: name of the field to mirror.
- `modifier`: allows transformation (e.g., numeric modification).
