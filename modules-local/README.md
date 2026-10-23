# modules-local

Your own Mathua modules. Everything in this folder except this README is
git-ignored, so nothing you put here is ever pushed.

A module is a folder with a `module.json`:

```
modules-local/
  wooldridge/
    module.json        {"id": "wooldridge", "name": "Wooldridge, Introductory Econometrics"}
    concepts/*.json    topics, same format as data/concepts
    bank/*.json        questions (code tasks, multiple choice, output, recall cards)
    readings.json      {"<concept id>": {"source": "Ch. 2.1, pp. 22-30", "file": "wooldridge.pdf", "page": 40}}
    files/             your PDFs; "Open section" opens them at the page above
```

To switch a module off without deleting it, list its id in
`modules.local.json` at the repository root:

```json
{"disabled": ["wooldridge"]}
```

Rebuild after any change: `docker compose up --build`. See FORK.md.
