#!/usr/bin/env bash

bold=$(tput -T xterm bold)
normal=$(tput -T xterm sgr0)

SESSIONS=$(sudo lsof -i :22 | wc -l)
TS_HOST=$(tailscale status | head -1 | awk '{print $2}')
echo " "
echo "connect: ${bold}ssh $(whoami)@${TS_HOST}${normal}"
echo " "

echo "waiting for a connection..."
journalctl -f -u ssh.service &
LOGGER=$!

until [[ "$(sudo lsof -i :22 | wc -l)" -gt "${SESSIONS}" ]]; do
    sleep 10s
done

echo "${bold}connected!${normal}"
echo "will stop after five minutes of inactivity"

INACTIVE=0
until [[ "${INACTIVE}" -ge "5" ]]; do
    if [[ "$(sudo lsof -i :22 | wc -l)" -le "${SESSIONS}" ]]; then
        ((INACTIVE++))
    else
        INACTIVE=0
    fi

    sleep 1m
done

kill "${LOGGER}"
echo "disconnected"
