# Copilot Instructions

## Code Style

- **NO DOCUMENTATION FILES**: Never create README, GUIDE, HOWTO, or any other documentation markdown files unless explicitly requested. Code changes only.
- **NO USELESS COMMENTS**: Don't add comments that just restate what the code does. If the code needs a comment to be understood, refactor it to be clearer instead.
- **Self-documenting code**: Use clear variable and function names. The code should explain itself.
- **Only comment WHY, not WHAT**: If you must add a comment, explain WHY something is done, not WHAT is being done.
- all terminal commands must work for zsh

### Bad Comments (Don't Do This)

```go
// Check what channels are actually available
signalLinked := false

// Build channel selector with only available options
var channelOptions string
```

### Good Comments (Rare, Only When Necessary)

```go
// HACK: QR service has 60s timeout, cache to avoid repeated calls
if time.Since(l.generatedAt) < l.ttl {
    return l.qrCode, nil
}
```

## General Rules

- Code must be idiomatic and follow Go best practices
- Keep functions focused and small
- Use clear, descriptive names
- Prefer composition over inheritance
- Handle errors properly, don't ignore them

