# Stability and bug fixes

This release resolves a comprehensive set of bugs identified in a static analysis and manual code review. All critical, high, medium, and low severity issues have been addressed.

## Critical fixes
- Fixed infinite recursion in `rotateWriter.Close()` — was calling itself instead of `w.fp.Close()`, crashing on every graceful shutdown
- Fixed nil map panic in `FileLogManager` — `filetracker` map was never initialized, making file logging completely non-functional
- Fixed nil map access in `FileLogManager.RegisterConfig()` — same root cause as above
- Fixed data race in signal replicator — `listen()` iterated over `signalChannels` without holding the read lock
- Fixed potential deadlock in signal replicator — sends to process channels are now non-blocking (`select` with `default`)
- Fixed multiple timer callbacks on repeated signals — `time.AfterFunc` is now guarded with `sync.Once`

## High severity fixes
- Fixed STDERR logging — was logging the full unsplit buffer for every line instead of the individual line
- Removed always-false nil guards on struct fields — `&field == nil` is never true; removed four dead code blocks and the unused `createLoggingConfig` helper
- Fixed watchdog goroutine never being started in `newRW()` — file rotation was silently not working
- Fixed panic on empty write in `BytePipe.Write()` — added length guard before accessing the last byte
- Fixed implicit nil error return in `BytePipe.Write()` — now returns explicit `nil` on success

## Medium severity fixes
- Fixed inverted logic in `Process.running()` — was returning `p.exited` (true when stopped) instead of `!p.exited`
- Fixed nil file handle after rename failure in `rotate()` — error is now logged and execution falls through to create a fresh file rather than returning with `w.fp = nil`
- Removed unused global variable `fileLogManager`
- Fixed goroutine leak in `LogManager.Submit()` — sending to a nil channel (missing map key) now guarded with an ok check
- Fixed `log.Fatalf` format string — was using string concatenation instead of a format argument (missing comma)
- Fixed data race on `p.exited` — assignment now wrapped in `Lock()`/`Unlock()` to match the `RLock()` in `running()`

## Low severity fixes
- Renamed misleading `rotateWriter.panic()` method to `logError()` — it never panicked
- Added safe ok-check for `*exec.ExitError` type assertion in `RunSecretProcess`
- Added comments in `Console.Submit()` explaining why write errors are intentionally not reported

## Tests
- Added regression test for `rotateWriter.Close()` to guard against infinite recursion
- Added regression test for `FileLogManager` nil map to guard against uninitialized map panics
- Added concurrent regression test for signal replicator to catch the data race under `-race`
