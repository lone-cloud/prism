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

## Design Context

### Users
Self-hosters — technically capable people who run their own infrastructure. They visit this UI occasionally to configure integrations (Signal, Telegram, WebPush, Proton Mail), register apps, and verify things are wired up. Could be on any device, but desktop is most common. Setup is a one-time or rare task; the UI is not used daily.

### Brand Personality
Sharp. Minimal. No-nonsense. This is a tool for people who run servers, not a product trying to sell them something. The interface should feel like it respects their time — no filler, no decorative chrome, no marketing energy.

### Aesthetic Direction
Refined utilitarian. Light theme by default, dark by system preference (keep existing both-modes behavior). The palette should feel clean and precise, not sterile. The cyan accent (`#56c3de`) is a good starting point for the brand hue — it reads as calm and technical without being loud. Neutrals should be very subtly tinted toward it.

Anti-reference: PHPMyAdmin / boring CRUD admin panel. The goal is something that wouldn't look out of place as a polished open-source project UI — functional, cohesive, with quiet confidence.

### Design Principles
1. **Earn every element** — if it doesn't serve a function, remove it. No decorative chrome.
2. **Precision over personality** — tight spacing, clear hierarchy, no ambiguity about what's interactive.
3. **Trust through clarity** — statuses should be instantly readable; nothing should make the user wonder "did that work?"
4. **Quiet confidence** — not boring, not flashy. Typography and spacing do the work, not color.
5. **Polish the structure, don't replace it** — the existing `<details>`-card pattern and HTMX architecture should be respected. Improve the CSS layer, don't overhaul templates.

