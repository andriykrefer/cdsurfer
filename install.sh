#!/usr/bin/env bash
set -e

shell_cfg_file="$HOME/.bashrc"
latest_release_file=https://github.com/andriykrefer/cdsurfer/releases/latest/download/cd-surfer_linux_amd64
fname="cd-surfer"

echo "Downloading latest release..."
rm -rf /tmp/$fname
wget -q -O /tmp/$fname $latest_release_file -O /tmp/$fname

echo "Installing in /bin/${fname}"
sudo chmod 755 /tmp/$fname
sudo mv /tmp/$fname /bin/

# Make a backup .bashrc
cp ${shell_cfg_file} "$shell_cfg_file.bk"

# Remove old config
sed '/# cd-surfer/d' "$shell_cfg_file.bk" > ${shell_cfg_file}

# Add config
printf 'function cds {                       # cd-surfer\n' >> ${shell_cfg_file}
printf '  eval "$(/bin/cd-surfer "$@")"    # cd-surfer\n' >> ${shell_cfg_file}
printf '}                                  # cd-surfer\n' >> ${shell_cfg_file}

# Reload the current shell
source $shell_cfg_file

echo "*************************************************************"
echo "* cd-surfer installed succcesfully!                         *"
echo "* Restart open terminals to take effect                     *"
echo "* Enter the command 'cds' in the terminal to call cd-surfer *"
echo "*************************************************************"
