#!/usr/bin/env bash

bold=$(tput -T xterm bold)
normal=$(tput -T xterm sgr0)

echo "::group::Starting funnel"
sudo tailscale funnel --tcp 8443 --bg --yes tcp://localhost:22
echo "::endgroup::"

SESSIONS=$(sudo lsof -i :22 | wc -l)

URL=$(tailscale funnel status --json | jq -r ".AllowFunnel | keys[0]")
DOMAIN=${URL%%:*}
PORT=${URL##*:}
echo " "
echo "Connect: ${bold}ssh -p ${PORT} $(whoami)@${DOMAIN}${normal}"
echo " "

journalctl -f -u ssh.service -o cat &
LOGGER=$!

until [[ "$(sudo lsof -i :22 | wc -l)" -gt "${SESSIONS}" ]]; do
    sleep 10s
done

echo "${bold}Connected!${normal} Will stop after five minutes of inactivity"

INACTIVE=0
until [[ "${INACTIVE}" -ge "5" ]]; do
    if [[ "$(sudo lsof -i :22 | wc -l)" -le "${SESSIONS}" ]]; then
        ((INACTIVE++))
        echo "Inactive for ${INACTIVE}/5 minutes"
    else
        INACTIVE=0
    fi

    sleep 1m
done

sudo tailscale funnel --tcp 8443 off
kill "${LOGGER}"
echo "disconnected"
