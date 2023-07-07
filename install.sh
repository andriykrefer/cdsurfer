#!/usr/bin/env bash

shell_cfg_file="$HOME/.bashrc"
lastes_release_file=https://github.com/andriykrefer/cdsurfer/releases/latest/download/cd-surfer_linux_amd64
fname="cd-surfer"

echo "Downloading latest release..."
sudo rm -rf /tmp/$fname
wget $lastes_release_file -O /tmp/$fname &>2 /dev/null

echo "Installing in /bin/${fname}"
sudo chmod a+x /tmp/$fname
sudo mv /tmp/$fname /bin/

# Make a backup .bashrc
cp ${shell_cfg_file} "$shell_cfg_file.bk"

# Remove old config
sed '/# cd-surfer/d' "$shell_cfg_file.bk" > ${shell_cfg_file}

# Add config
printf '\nalias s="eval \$(/bin/cd-surfer)" # cd-surfer\n' >> ${shell_cfg_file}

# Reload the current shell
source $shell_cfg_file

echo "*************************************************************"
echo "* cd-surfer installed succcesfully!                         *"
echo "* Restart open terminals to take effect                     *"
echo "* Enter the command 's' in the terminal to call cd-surfer   *"
echo "*************************************************************"
