# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- 3-panel detail layout (wide terminals): feed list on the left, metadata
  panel (source, full timestamp, seen status, link) on the top-right, and
  scrollable markdown description panel on the bottom-right. Falls back to
  the single-panel table on narrow terminals.

- `ctrl+h` / `ctrl+l` to switch focus between the list and description
  panels (ecosystem convention from taglue/tabelharadar). Description panel
  captures j/k/pgup/pgdn/home/end when focused.

- `body` field now rendered as styled markdown in the description panel
  via `tabelhatuiui v0.7.0`'s `markdown.Panel` + glamour.
