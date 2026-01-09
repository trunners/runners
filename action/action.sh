#!/usr/bin/env bash

bold=$(tput -T xterm bold)
normal=$(tput -T xterm sgr0)

# setup shell

if [[ "${RUNNER_OS}" == "Linux" ]]; then
    echo "setting default shell to bash"
    sudo chsh -s "$(which bash)" "$(whoami)"

elif [[ "${RUNNER_OS}" == "macOS" ]]; then
    echo "setting default shell to zsh"
    sudo chsh -s "$(which zsh)" "$(whoami)"
fi

# setup ssh

echo "::group::overwriting sshd_config"
cat "${GITHUB_WORKSPACE}/action/sshd_config" | sudo tee /etc/ssh/sshd_config > /dev/null
sudo cat /etc/ssh/sshd_config
echo "::endgroup::"

echo "setting ssh keys"
echo "${PRIVATE_KEY}" | sudo tee /etc/ssh/ssh_host_ed25519_key > /dev/null
echo "${PUBLIC_KEY}" | sudo tee /etc/ssh/ssh_host_ed25519_key.pub > /dev/null

echo "::group::adding ${GITHUB_ACTOR}'s keys to authorized_keys"
curl -s "https://github.com/${GITHUB_ACTOR}.keys" | sudo tee /etc/ssh/authorized_keys > /dev/null
sudo cat /etc/ssh/authorized_keys
echo "::endgroup::"

echo "starting ssh service"
if [[ "${RUNNER_OS}" == "Linux" ]]; then
    sudo systemctl start ssh.service
elif [[ "${RUNNER_OS}" == "macOS" ]]; then
    sudo systemsetup -setremotelogin on
fi

# setup tailscale funnel

echo "::group::Starting funnel"
sudo tailscale funnel --tcp 8443 --yes --bg tcp://localhost:22
echo "::endgroup::"

SESSIONS=$(sudo lsof -i :22 | wc -l)
URL=$(tailscale funnel status --json | jq -r ".AllowFunnel | keys[0]")
DOMAIN=${URL%%:*}
PORT=${URL##*:}
echo " "
echo "Connect: ${bold}ssh -p ${PORT} $(whoami)@${DOMAIN}${normal}"
echo " "

# log ssh connections

if [[ "${RUNNER_OS}" == "Linux" ]]; then
    journalctl -f -u ssh.service -o cat &
elif [[ "${RUNNER_OS}" == "macOS" ]]; then
    sudo log stream --predicate 'process == "sshd"' &
fi
LOGGER=$!

until [[ "$(sudo lsof -i :22 | wc -l)" -gt "${SESSIONS}" ]]; do
    sleep 10s
done

echo "${bold}Connected!${normal} Will stop after five minutes of inactivity"

# wait for inactivity

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

# teardown

sudo tailscale funnel --tcp 8443 --yes off
kill "${LOGGER}"
