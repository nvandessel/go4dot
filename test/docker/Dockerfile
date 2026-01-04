FROM ubuntu:latest

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install dependencies
RUN apt-get update && apt-get install -y \
    git \
    stow \
    curl \
    sudo \
    zsh \
    locales \
    vim \
    neovim \
    tmux \
    tree \
    && rm -rf /var/lib/apt/lists/*

# Set up locales for UTF-8 support (essential for TUI)
RUN locale-gen en_US.UTF-8
ENV LANG en_US.UTF-8
ENV LANGUAGE en_US:en
ENV LC_ALL en_US.UTF-8
ENV TERM xterm-256color

# Install g4d binary
COPY g4d /usr/local/bin/g4d
RUN chmod +x /usr/local/bin/g4d

# Copy examples to a staging location (will be moved to user home later)
COPY examples /opt/examples

# Create a non-root user with sudo privileges
RUN useradd -m -s /bin/zsh testuser && \
    echo "testuser ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Set up user environment
USER testuser
WORKDIR /home/testuser

# Copy examples to user home directory
RUN cp -r /opt/examples ~/examples

# Configure ZSH
RUN echo 'autoload -U colors && colors' > .zshrc && \
    echo 'PROMPT="%{$fg[green]%}testuser@sandbox%{$reset_color%}:%{$fg[blue]%}%~%{$reset_color%}$ "' >> .zshrc && \
    echo 'alias ll="ls -la"' >> .zshrc && \
    echo 'alias reset-examples="rm -rf ~/examples && cp -r /opt/examples ~/examples && echo \"Examples reset to original state\""' >> .zshrc

# Default command
CMD ["/bin/zsh"]
