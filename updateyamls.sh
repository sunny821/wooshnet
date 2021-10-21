#!/bin/sh

SHELL_FOLDER=$(dirname "$0")

sh ${SHELL_FOLDER}/tools/yamls_wsl.sh
sh ${SHELL_FOLDER}/tools/yamls_dev105.sh

