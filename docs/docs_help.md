# Help with Arlon Documentation

## Prerequisites

Documentation for Arlon is written in Markdown format, and generated using the [mkdocs](https://www.mkdocs.org) site generator.
The theme mkdocs-material has been used, and docs from the main branch are published to a `GitHub pages` site.

- Install Python 3.x (3.8 or later recommended. 3.11 has been tested)
- Install the pip python package manager for Python 3.
- Install the Python modules mentioned in docs/requirements.txt
- Run `mkdocs serve` to test any local changes to docs, and commit them using git workflow

## Help with MkDocs

Welcome to Arlon documentation with MkDocs

For full documentation, visit [mkdocs.org](https://www.mkdocs.org).

### Commands

- `mkdocs new [dir-name]` - Create a new project.
- `mkdocs serve` - Start the live-reloading docs server.
- `mkdocs build` - Build the documentation site.
- `mkdocs -h` - Print help message and exit.

## Project layout

    mkdocs.yml    # The configuration file. 
                  # Contains the navigation structure 
    docs/
        index.md  # The documentation homepage.
        ...       # Other markdown pages, images and other files.
