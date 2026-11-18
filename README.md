# playhold-tictactoe

The code of [playhold](https://github.com/ny4rl4th0t3p/playhold)'s tutorial: tic-tac-toe, built one chapter at a time.
The tutorial itself is in playhold's [documentation](https://ny4rl4th0t3p.github.io/playhold/); this repository holds
each chapter's finished code, so you can compare yours with it, or run it.

| Folder      | What the game has by the end of the chapter                         |
|-------------|---------------------------------------------------------------------|
| `chapter1/` | A module: the board's state, a move, a view for each player, tests |
| `chapter2/` | Its own messages, in a `.proto` generated with buf                  |
| `chapter3/` | A server, and a test that plays a game through the Go SDK           |
| `chapter4/` | A page in the browser, on the TypeScript SDK, hosted on its own     |
| `chapter5/` | Saving in PostgreSQL, so a game survives the server stopping        |
| `chapter6/` | A time limit on each turn, and a saved state's new version          |

Each folder is a complete, tested state of the game, in one Go module, so two chapters can be compared with `diff`.
It builds on playhold's releases as any game would: playhold at the version in `go.mod`. A version `v0.X.y` of this
tutorial is built on playhold `v0.X`.

```bash
make check   # build, vet, playhold's determinism checks, tests, buf lint, pages and vuln, for every chapter
make gen     # generate every chapter's code from its .proto files: Go, and the pages' TypeScript
```

## License

[Apache-2.0](LICENSE).