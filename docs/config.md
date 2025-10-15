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
  "statement": "SELECT customerid, amount, customer_email FROM account.customer;"
}
```

The cache can alternatively be populated using an sql statement sourced from a file using the `source` config attribute. The path should be relative to the execution directory.

```json
"cache": {
  "source": "test/customer_cache.sql"
}
```

The output file can also be seeded from an existing CSV file. This works the same as seeding from a database. Aliases can be used in the same manner.

```json
"cache": {
  "source": "test/existing.csv"
}
```

Fields can also be seeded with a CSV cache specific to the given field. This is useful for adhoc injection from sources outside of usual operation. The injection can optionally be governed by a `rate` which is the precentage chance of the cache being used. If it misses it will fallback to the generic cache if enabled, and finally to generation logic.

```json
{ "name": "id", "type": "uuid", "source": "source/id_subset.csv", "rate": "50", "alias": "customerid"}
```


## Connection Profiles
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

### Seed

The file config can optionally be provisioned with a seed. This will be the seed used for all RNG operations in the generation, giving deterministic results.

```json
"config": {
  "file_name": "testfile.csv",
  "delimiter": "|",
  "row_count": 6,
  "include_headers": true,
  "seed": "47e7f672-9c3d-4dd4-a151-6f5fd67f236f"
},
```

Any run without a seed will output the seed used in generation to the console, which can be used to replicate outputs.

---

## Postprocessing
A `postprocessing` block can be provided in the json config to allow you to upload files generated directly to an s3 location. In the future this will also support encryption. To authenticate the upload you must be authenticated against the destination account. This will allow the tool to leverage your token in `~/.aws/credentials`. A working example of the config block can be seen below. The file name will be concatenated to the location, landing in a directory at the given location.

```json
  "postprocess": {
    "enabled": true,
    "location": "s3://{BUCKET_NAME}/{PATH}/{PREFIX}",
    "region": "eu-west-2"
  },
```

## Field Types

When configuring your CSV generation, each field in the `fields` array represents a column with specific data logic. The name provided will be the name of the column in the output file.

Any field can be seeded from the database or an existing CSV using the following syntax. If you want to have the column name in the ouput differ from the column name of the database, you can pass an alias. This will be the database column, and the name will be the csv column.

```json
{ ... "name":"id", "seed":true, "alias":"customerid" }
```

---
### Supported field types:
---

### `Override`

This will hardcode a value to the given input. The row will always contain the supplied value.

```json
{ "name": "active", "value": "true" }
```

This can also be used to create whitespace values in the output csv by providing an empty string for input.

```json
{ "name": "active", "value": "" }
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

Generates a random floating-point number between a minimum and maximum. An optional format can be passed to specify the number of decimal places. This will default to 0 if not provided.

```json
{ "name": "amount", "type": "number", "format": "2", "min": -2000.00, "max": 2000.00 },
```

If you need to generate a random number of a given length, you can pass a `length` attribute. This will generate a random number with a fixed number of digits.

```json
{ "name": "code", "type": "number", "length": 14 },
```

---

### `timestamp`

Creates a timestamp using the current time formatted with Go-style time syntax. You can optionally pass an interval to offset the time. This is provided as seconds and supports both positive and negative values.  This is not currently deterministic.

```json
{ "name": "updated_at", "type": "timestamp", "interval": -604800 , "format": "02-01-06 15:04:05" }
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

Copies the value of another field. Can optionally modify numeric inputs by supplying a `modifier`. The target will be multiplied by the modifier.

```json
{ "name": "inverse", "type": "reflection", "target": "amount", "modifier": -1 }
```

- `target`: name of the field to mirror.
- `modifier`: allows transformation (e.g., numeric modification).

---
## Functions
Function strings can be used to drive how a supported faker generates its values. This can be used to create psuedo-trends accross the output file

**How it works**

The function string is parsed into a `name` and a `params` map.

A shared sampler produces a normalized value norm ∈ [0,1] for the chosen name (e.g. sin, random, exponential, linear, constant). Time-based functions accept a period.

The normalized value is mapped to the field type:

**Number:** norm → numeric value via MapNormalizedToFloat(norm, params, min, max).

**Timestamp:** norm → duration offset via MapNormalizedToDuration(norm, params, interval, dir) and added to now.

Optional modifiers (amplitude, center, clamp, jitter, etc.) alter sampling or mapping.


### Supported functions

`random` — uniform random in [0,1].

`constant` — returns a fixed value (valuenorm in [0,1] or value as an absolute number/duration).

`sin` — sinusoidal wave (time-based); accepts period and phase.

`linear` — repeating linear ramp (sawtooth) over period.

`exponential` — heavy-tailed generator; accepts scale and side.

### Supported modifiers

`period` — seconds (numeric) or duration string (7d, 72h, 1.5d) — used by sin/linear.

`phase` — degrees (for sin).

`dir` — for timestamps: future (default) | past | both. If omitted, a negative interval implies past.

`interval` — base magnitude for timestamps (duration string like 7d or numeric seconds). Top-level interval (field root) is still supported for backwards compatibility. This will be deprecated in the near future.

`amplitude` — multiplier applied to base magnitude or half-range (default 1.0).

`center` — shifts the midpoint:`

`clamp` — "true" (default) or "false". When false, mapped results may exceed [min,max] (for numbers) or the base timestamp window.


### Jitter / outlier params
You can also pass `jitter` paramaters to generate outliers outside of the normal function declaration. The rate of jitter generation can be controlled with the following parameters.

`jitter` — probability (0..1) of producing an outlier on a Generate() call.

`jitter_type` — scale | edge | spike | exponential.

`jitter_amp` — multiplier used by scale and exponential (default 3.0).

### Supported jitter types
**scale:** multiplies the normalized value (pushes toward an edge).

**edge:** returns exactly 0 or 1.

**spike:** small spikes near an edge (e.g., [0,0.1] or [0.9,1]).

### Function Examples ###
There is a sample config file demonstrating what can be done with functions in at the `/docs/functions.json` path in this repository as well as the sample fields provided below.

Uniform random timestamp in the past week
```
  { "type": "timestamp", "format": "2006-01-02", "function": "random:dir=past,interval=7d" }
```

All rows same: exactly one year in the past
```
  { "type": "timestamp", "format": "2006-01-02", "function": "constant:dir=past,interval=52w" }
```
Sin wave mapped to number range, small jitter outliers
```
  { "type": "number", "min": 0, "max": 10000, "function": "sin:period=7d,amplitude=1.5,center=50,jitter=0.005,jitter_type=scale,jitter_amp=3" }
```

Heavy-tailed main generator, allow overshoot beyond bounds
```
  { "type": "number", "min": 0, "max": 1000, "function": "exponential:scale=3,side=high,clamp=false" }
```

Constant 3 days ago (timestamp)
```
  { "type": "timestamp", "format":"2006-01-02", "function":"constant:value=3d,dir=past" }
```