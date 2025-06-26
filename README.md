# Branch Sweeper CLI

## Contents

- [Description](#description)
  - [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
  - [Commands](#commands)
  - [Examples](#examples)
- [Contributing](#contributing)
- [License](#license)

## Description

Branch Sweeper CLI is a tool for identifying and removing stale Git branches across multiple local repositories.

### Features

- **List stale branches:** Scan one or more directories and output stale branches older than a given number of days.
- **Prune stale branches:** Delete branches that meet the stale criteria.

## Installation

To install the tool, follow this steps:

```bash
curl -s -L https://github.com/byFrederick/branch-sweeper/releases/download/{version}/branch-sweeper_{os}_{arch}.tar.gz | tar xz
chmod +x branch-sweeper
sudo mv branch-sweeper /usr/local/bin
```

## Usage

After installation, the `branch-sweeper` command is available to manage stale branches.

### Commands

- `list`: Display stale branches without deleting them.
- `prune`: Delete stale branches.

Global flags apply to both commands:

- `--base, -b`: Repository base branch (default `main`).
- `--days, -d`: Minimum days since last commit to mark a branch stale (default `30`).
- `--exclude, -e`: Glob pattern for branches to exclude (use braces for multiple patterns, e.g. '{feat*,fix*}').
- `--include, -i`: Glob pattern for branches to include (use braces for multiple patterns, e.g. '{feat*,fix*}').
- `--merged, -m`: Include branches already merged into the base branch.
- `--path, -p`: Directory to scan for Git repos (default `.`).

### Examples

List stale branches older than 60 days:

```bash
branch-sweeper list --days 60 --path ~/projects
```

Delete merged branches older than 90 days:

```bash
branch-sweeper prune --merged --days 90 --path ~/projects
```

## Contributing

If you want to contribute, follow these steps:

1. Fork the repository.
2. Create a new branch (`git checkout -b feature-branch`).
3. Make your changes.
4. Commit your changes (`git commit -m 'Add new feature'`).
5. Push to the branch (`git push origin feature-branch`).
6. Open a Pull Request.

## License

This project is licensed with the [MIT license](LICENSE).
