# Rapid Downloader 

## Prerequisites
- [Pyenv](https://github.com/pyenv/pyenv?tab=readme-ov-file#installation)
- [Poetry](https://python-poetry.org/docs/#installation)

## Setup
Install the requirements
```bash
pyenv install 3.12
pyenv local 3.12
poetry env use $(pyenv which python)
poetry install
```

## Development
```
poetry poe dev
```

### Build
```
poetry poe build
```
