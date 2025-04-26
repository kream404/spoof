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

---


## Documentation

- [Configuration Documentation](./docs/config.md)

## Examples

Generate a CSV using a config file:

```bash
spoof -c ./configs/sample.json -V
```
---

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
