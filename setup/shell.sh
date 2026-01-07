#!/usr/bin/env bash

if [[ "${RUNNER_OS}" == "Linux" ]]; then
    echo "setting default shell to bash"
    sudo chsh -s "$(which bash)" "$(whoami)"

elif [[ "${RUNNER_OS}" == "macOS" ]]; then
    echo "setting default shell to zsh"
    sudo chsh -s "$(which zsh)" "$(whoami)"
fi
