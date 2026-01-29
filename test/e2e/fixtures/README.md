# E2E Test Fixtures

This directory contains test fixtures used by E2E tests.

## Structure

- `dotfiles/` - Minimal test dotfiles repository
  - `.go4dot.yaml` - Test configuration
  - `vim/` - Test vim configuration
  - `zsh/` - Test zsh configuration

## Usage

These fixtures are copied into Docker containers during E2E testing to provide
a consistent, minimal dotfiles setup for testing g4d functionality.

## Maintenance

Keep these fixtures as minimal as possible while still testing core functionality.
If you need to test specific features, create additional fixture directories rather
than complicating these.
