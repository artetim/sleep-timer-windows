# Changelog

## v1.0.1 (2026-11-13)

- Reduced active-countdown flicker by painting through an offscreen buffer.
- Repaints only the countdown area when the displayed second changes.
- Avoids redrawing native input and button controls unnecessarily.
- Preserves the existing English interface, icon, duration presets, and Windows sleep command.
- Automated tests passed. Real Windows verification of this exact build is still pending.

## v1.0.0 (2026-11-13)

- Refined the interface layout, buttons, spacing, and status messages.
- Added the app icon to the executable and title bar.
- Maintained the single-purpose sleep timer design.
- The project owner tested a previous build on Windows 10 and reported successful sleep behavior, plus visual flickering when the countdown was active.
