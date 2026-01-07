#!/usr/bin/env bash

bold=$(tput -T xterm bold)
normal=$(tput -T xterm sgr0)

SESSIONS=$(sudo lsof -i :22 | wc -l)
TS_HOST=$(tailscale status | head -1 | awk '{print $2}')
echo " "
echo "Connect: ${bold}ssh $(whoami)@${TS_HOST}${normal}"
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

kill "${LOGGER}"
echo "disconnected"
