#!/usr/bin/env bash

if [[ "${RUNNER_OS}" == "Linux" ]]; then
    echo "::group::installing kitty terminfo"
    sudo apt-get install -y kitty-terminfo
    echo "::endgroup::"

    echo "setting default shell to bash"
    sudo chsh -s "$(which bash)" "$(whoami)"
    
elif [[ "${RUNNER_OS}" == "macOS" ]]; then
    brew install --cask kitty
fi
