# Spoof

**Spoof** is a command-line tool for generating CSV files with flexible, configurable data logic — including support for database-backed seeding and fake data generation.

---
## Features
- Deterministic Generation
- Generate structured CSV files based on JSON config.
- Optionally seed data from DB
---
## Installation

Currently only Mac and Linux systems are supported. Windows users can install via WSL. To install run the command below.

```bash
git clone https://github.com/kream404/spoof.git && cd spoof && ./install.sh
```

---

## Usage

```bash
spoof [flags]
```

---

## Flags

| Flag                      | Shorthand | Description                                         |
|---------------------------|-----------|-----------------------------------------------------|
| `--version`               | `-v`      | Show CLI version.                                   |
| `--verbose`               | `-V`      | Output detailed logs and execution time.            |
| `--config <path>`         | `-c`      | Path to JSON configuration file.                    |
| `--profile <name>`        | `-p`      | Name of DB connection profile (overrides config).   |
| `--generate`               | `-g`      | Generate a new config file.                                   |
| `--extract <path>`               | `-e`      | Extract a config file from a csv                                   |

---


## Documentation

- [Configuration Documentation](./docs/config.md)

## Examples

Generate a CSV using a config file:

```bash
spoof --config ./configs/sample.json
```
---

## Extraction

You can also generate a config file from a CSV. The type inference is not perfect and may take some additional tweaking after generation, but it should be a good starting point. This is a work in progres.

> If the target CSV file does not have headers, you **must** annotate the CSV with headers. These headers will be the `name` of the `field` in the generated config file. This will output the generated file in your current working directory

```bash
spoof --extract ./path/to/csvfile.csv
```

## Connection Profiles

Profiles are stored at `~/.config/spoof/profiles.ini`. Example:

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

## Output

CSV files are saved to:

```
./output/output.csv
```

This will make an output directory in the execution directory if one does not exist.

---
