#!/bin/sh

export varnamDir=$(dirname "$(readlink -f "$0")")
export LD_LIBRARY_PATH="$varnamDir:$LD_LIBRARY_PATH"

if [ ! -d /usr/local/share/varnam/vst ] && [ ! -d /usr/share/varnam/vst ] && [ ! -d schemes ] && [ ! -d ~/.local/share/varnam/vst ]; then
  mkdir -p ~/.local/share/varnam/vst
fi

$varnamDir/varnam --config $varnamDir/config.toml
