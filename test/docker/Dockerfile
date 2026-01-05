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
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Set up locales for UTF-8 support (essential for TUI)
RUN locale-gen en_US.UTF-8
ENV LANG en_US.UTF-8
ENV LANGUAGE en_US:en
ENV LC_ALL en_US.UTF-8
ENV TERM xterm-256color

# Build arguments
ARG NO_INSTALL=false
ARG DOTFILES_URL=""
ARG NO_EXAMPLES=false

# Environment variables
ENV NO_INSTALL=$NO_INSTALL
ENV DOTFILES_URL=$DOTFILES_URL
ENV NO_EXAMPLES=$NO_EXAMPLES

# Copy g4d binary if not in no-install mode
COPY g4d* /tmp/
RUN if [ "$NO_INSTALL" != "true" ]; then \
        mv /tmp/g4d /usr/local/bin/g4d && \
        chmod +x /usr/local/bin/g4d; \
    fi && rm -f /tmp/g4d*

# Copy install script for manual testing
COPY install.sh /opt/install.sh
RUN chmod +x /opt/install.sh

# Copy examples to a staging location
COPY examples /opt/examples

# Create a non-root user with sudo privileges
RUN useradd -m -s /bin/zsh testuser && \
    echo "testuser ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Set up entrypoint
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Set up user environment
USER testuser
WORKDIR /home/testuser

# Configure ZSH
RUN echo 'autoload -U colors && colors' > .zshrc && \
    echo 'PROMPT="%{$fg[green]%}testuser@sandbox%{$reset_color%}:%{$fg[blue]%}%~%{$reset_color%}$ "' >> .zshrc && \
    echo 'alias ll="ls -la"' >> .zshrc && \
    echo 'alias reset-examples="rm -rf ~/examples && cp -r /opt/examples ~/examples && echo \"Examples reset to original state\""' >> .zshrc

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["/bin/zsh"]
