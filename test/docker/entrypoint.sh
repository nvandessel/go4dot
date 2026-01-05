#!/bin/zsh
set -e

# Configure git safety
git config --global --add safe.directory /home/testuser/dotfiles

# Clone dotfiles if URL is provided
if [ -n "$DOTFILES_URL" ]; then
    echo "Cloning dotfiles from $DOTFILES_URL..."
    git clone "$DOTFILES_URL" ~/dotfiles
fi

# Copy examples if not disabled
if [ "$NO_EXAMPLES" != "true" ]; then
    echo "Populating ~/examples..."
    cp -r /opt/examples ~/examples
fi

# Welcome message
echo "--------------------------------------------------------"
echo "Welcome to the go4dot sandbox!"
if [ "$NO_INSTALL" = "true" ]; then
    echo "Mode: No-Install (g4d is NOT pre-installed)"
    echo "To install g4d, run: bash /opt/install.sh"
else
    echo "Mode: Standard (g4d is pre-installed)"
    g4d version
fi
echo "--------------------------------------------------------"

exec "$@"
