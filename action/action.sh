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

# setup client

if [[ "${RUNNER_OS}" == "Linux" ]]; then
    RELEASE_OS="linux"
elif [[ "${RUNNER_OS}" == "macOS" ]]; then
    RELEASE_OS="darwin"
fi

if [[ "${RUNNER_ARCH}" == "X64" ]]; then
    RELEASE_ARCH="amd64"
elif [[ "${RUNNER_ARCH}" == "ARM64" ]]; then
    RELEASE_ARCH="arm64"
fi

echo "::group::installing client"
gh release download --repo trunners/runners --pattern "client_*_${RELEASE_OS}_${RELEASE_ARCH}" --output runner
chmod +x runner
echo "::endgroup::"

echo "starting client"
./runner &
CLIENT=$!

SESSIONS=$(who | grep -c -e pts -e tty)

echo " "
echo "Ready to connect via SSH!"
echo " "

# log ssh connections

if [[ "${RUNNER_OS}" == "Linux" ]]; then
    journalctl -f -u ssh.service -o cat &
elif [[ "${RUNNER_OS}" == "macOS" ]]; then
    sudo log stream --predicate 'process == "sshd"' &
fi
LOGGER=$!

# wait for connection

until [[ "$(who | grep -c -e pts -e tty)" -gt "${SESSIONS}" ]]; do
    sleep 5s
done

echo "${bold}Connected!${normal}"

# wait for disconnection

until [[ "$(who | grep -c -e pts -e tty)" -le "${SESSIONS}" ]]; do
    sleep 5s
done

# teardown
kill "${LOGGER}"
kill "${CLIENT}"
