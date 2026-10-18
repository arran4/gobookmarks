1. **Extract Conversion Logic**: Move the core conversion logic inside `ConvertCommand.Execute` into a reusable function, e.g., `Convert(fromFormat, toFormat string, r io.Reader, w io.Writer) error`.
2. **Inject IO into ConvertCommand**: Add `Stdin io.Reader` and `Stdout io.Writer` fields to `ConvertCommand`. Update `NewConvertCommand` to default to `os.Stdin` and `os.Stdout`. Modify `Execute` to read from the provided `Stdin` (or file if a path is passed) and write to `Stdout`, rather than directly mutating or relying entirely on `os.Stdin`/`os.Stdout`.
3. **Refactor Convert Tests**:
   - Split `TestConvertCommand` into smaller, focused tests:
     - `TestConvertLogic`: A table-driven test calling the extracted `Convert` function directly using in-memory readers (`strings.Reader`) and writers (`bytes.Buffer`). This will cover text-to-JSON, JSON-to-text, unnamed tab handling, and lossy JSON failures without touching the filesystem.
     - `TestConvertCommand_CLI`: A smaller test focusing on CLI integration (filename handling, stdin fallback, and stdout wiring), using injected `Stdout` rather than `os.Stdout` patching.
   - Use small `mustWriteFile` helpers where temp files are necessary (e.g., in `TestConvertCommand_CLI`).
4. **Pre-commit Checks**: Follow required pre-commit instructions, run `go test`, lint, etc.
