#!/usr/bin/env bash

echo "::group::overwriting sshd_config"
cat "${GITHUB_WORKSPACE}/sshd_config" | sudo tee /etc/ssh/sshd_config > /dev/null
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
