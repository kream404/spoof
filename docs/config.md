# Configuration

## Top-Level Structure
```
{
  "config": { ... },
  "cache": { ... },
  "fields": [ ... },
  "postprocess": { ... }
}
```
---

# Execution Model

Each field is evaluated in the following order:

1. Source Injection
2. Cache Seeding
3. Fallback Generation

This applies to both top-level and nested JSON fields.

---

# Bundles

Bundles allow you to orchestrate sequential file generation. This facilitate complex data generation as we can seed directly from the ouput of previous steps. Best practice would be to have a `bundles` and `configs` directory at the same level and give a relative path to the config

## Usage

```bash
spoof -c <path_to_bundle>
```

### Example

```bash
{
  "files": [
    {
      "source": "configs/customers.json"
    },
    {
      "source": "configs/bankaccount.json"
    },
    {
      "source": "configs/cards.json"
    },
    {
      "source": "configs/balance.json"
    }
  ]
}
```

---

# Inspect Mode

Inspect mode provides a dry-run analysis of a bundle or config without executing any generation.

## Usage

```bash
spoof --inspect -c <path> [-i VAR=value,...]
```

### Example

```bash
spoof --inspect -c bundles/bundle.json
```

## What Inspect Does

- Traverses bundles and configs recursively
- Resolves file paths using injected variables
- Detects unresolved and missing variables
- Builds an execution plan
- Aggregates variable metadata from:
  - inline template usage (`{{ VAR }}`)
  - declared `variables` blocks in bundles/configs
- Distinguishes required vs optional variables
- Suggests a runnable command

# Connection Profiles
Connection profiles can be used to quickly seed data from different environments. Profiles are stored at `~/.config/spoof/profiles.ini` and contain connection variables for a specified environment.

```ini
[local]
hostname = localhost
port = 5432
username = user
password =
name = mydb
```

> If the `password` field is empty, you will be prompted at runtime.


To use a database profile override, you can pass the profile as an argument:

```bash
spoof -c ./configs/sample.json -p local
```
---

# Cache Configuration

Defines how seed data is loaded.

## Database
```
"cache": {
  "hostname": "localhost",
  "port": "5432",
  "username": "user",
  "password": "password",
  "name": "database",
  "statement": "SELECT customerid, amount FROM account.customer;"
}
```
## SQL File
```
"cache": {
  "source": "test/customer_cache.sql"
}
```
## CSV
```
"cache": {
  "source": "test/existing.csv"
}
```
## S3
```
"cache": {
  "source": "s3://bucket/path/",
  "region": "eu-west-2"
```
---

# Seed Selector

Lookup-based seeding instead of positional indexing.
```
"cache": {
  "source": "output/companies.csv",
  "selector": { 
    "column": "name",
    "keys": [
      "c1",
      "c2",
      "c3"
    ]
  }
}
```

This will inject the value at a given column when `"selector": true` is applied to a field

```
{ 
  "name": "customerid", 
  "seed": true, 
  "selector": true,
  "alias": "id"
},
```
## Behaviour

- Picks key per row
- Finds matching cache row
- Returns value from requested column

## Errors

- No match found
- Multiple matches found
- Output column missing

---

# Field Configuration
```
{
  "name": "id",
  "type": "uuid",
  "seed": true,
  "alias": "customerid"
}
```
## Name vs Alias

- name → output column
- alias → lookup column (cache/source)

---

# Source Injection
```
{
  "name": "id",
  "type": "uuid",
  "source": ["data.csv", "backup.json"],
  "rate": 50
}
```
## Behaviour

- Multiple sources are merged
- Supports CSV and JSON
- Indexed via seedIndex

## Conditions

Injection only occurs if:

- Source exists
- File type is supported (.csv, .json)
- Rate check passes

---

## Rate

Controls probability of injection or seeding. If not provided it will seed at a 100% rate. If a rate is supplied, fallback configuration must be provided. This will determine the generation if it is a "cache miss"
```
{
  "name": "code",
  "seed": true,
  "alias": "processcode",
  "rate": "50",
  "type": "number",
  "length": 4
},
```
## Behaviour

- 0 → never inject
- 100 → always inject
- null → defaults to 100
- Uses RNG comparison

---

# Field Types

## Override
```
{ "name": "active", "value": "true" }
```
## Iterator
```
{ "name": "id", "type": "iterator", "start": 100 }
```
## UUID
```
{ "name": "id", "type": "uuid" }
```
## Alphanumeric
```
{ "name": "code", "type": "alphanumeric", "regex": "^A[0-9]{5}$" }
```
## Range
```
{ "name": "status", "type": "range", "values": "1,2,3" }
```
## Foreach
```
{ "name": "status", "type": "foreach", "values": "A,B,C" }
```
Sequentially cycles values per row.

## Number
```
{ "name": "amount", "type": "number", "min": 0, "max": 100 }
```
Supports:
- min/max range
- length-based generation
- decimal formatting

## Timestamp
```
{
  "name": "created",
  "type": "timestamp",
  "format": "2006-01-02"
}
```
Supports:
- interval offsets
- function-driven generation

## Email
```
{ "name": "email", "type": "email" }
```
---

## Reflection
```
{
  "name": "inverse",
  "type": "reflection",
  "target": "amount",
  "modifier": -1
}
```
## Behaviour

- Reads from:
  1. Current generated fields
  2. Parent (for nested JSON)

- Modifier multiplies numeric values

---

# JSON Fields

## Template Example
```
{
  "id": "${id:string}",
  "amount": "${amount:number}"
}
```
## Field Config
```
{
  "name": "payload",
  "type": "json",
  "template": "template.json",
  "fields": [ ... ]
}
```
---

## JSON Root Types

### Object Root

Produces:
```
{ "id": "...", "amount": 10 }
```
---

### Array Root

If template starts with '[':
```
[
  { "id": "${id}" }
]
```
## Repeat
```
{ "repeat": "3" }
```
Produces:

[
  {...},
  {...},
  {...}
]

---

## Nested Evaluation

- Each JSON field runs in its own evaluation context
- Parent fields are accessible (for reflection)

---

## Modifier
```
{
  "name": "double",
  "type": "reflection",
  "target": "amount",
  "modifier": 2
}
```
## Behaviour

- Multiplies numeric values
- Preserves decimal precision

---

# Functions

Functions shape generated values.

## Format
```
name:param=value,param=value
```
---

## Supported Functions

random — uniform distribution

constant — fixed value

sin — sinusoidal pattern

linear — ramp pattern

exponential — heavy-tailed distribution


# Function Parameters

- period (time duration or seconds)
- phase (degrees)
- amplitude
- center
- clamp (true/false)
- dir (past, future, both)
- interval (duration)

---

# Jitter

Introduces outliers.
```
{
  "jitter": 0.01,
  "jitter_type": "scale"
}
```
## Types

- scale
- edge
- spike
- exponential

---

# JSON Output Behaviour

## Single JSON Field Optimization

If only one JSON field exists:
```
{ "payload": {...} }
```
Output becomes:

{ ... }

---

# Postprocessing

## S3 Upload
```
"postprocess": {
  "enabled": true,
  "location": "s3://bucket/path",
  "region": "eu-west-2"
}
```
---

## Database Insert
```
"postprocess": {
  "operation": "insert",
  "location": "database"
}
```
---

## Database Delete
```
"postprocess": {
  "operation": "delete",
  "key": "id",
  "type": "uuid"
}
```
---

## Deterministic Runs
```
"config": {
  "seed": "uuid"
}
```
- Ensures reproducible output
- Seed is generated and printed if not provided

---