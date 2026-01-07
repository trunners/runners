#!/usr/bin/env bash

if [[ "${RUNNER_OS}" == "Linux" ]]; then
    # Add cloudflare gpg key
    sudo mkdir -p --mode=0755 /usr/share/keyrings
    curl -fsSL https://pkg.cloudflare.com/cloudflare-public-v2.gpg | sudo tee /usr/share/keyrings/cloudflare-public-v2.gpg >/dev/null

    # Add this repo to your apt repositories
    echo 'deb [signed-by=/usr/share/keyrings/cloudflare-public-v2.gpg] https://pkg.cloudflare.com/cloudflared any main' | sudo tee /etc/apt/sources.list.d/cloudflared.list

    # install cloudflared
    sudo apt-get update && sudo apt-get install cloudflared

elif [[ "${RUNNER_OS}" == "macOS" ]]; then
    brew install cloudflared
else
    echo "Unsupported OS: ${RUNNER_OS}"
    exit 1
fi

sudo cloudflared service install "${CLOUDFLARE_TOKEN}"
