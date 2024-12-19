#!/bin/bash

# Declare site in YAML, as documented on the documentation: https://help.alwaysdata.com/en/marketplace/build-application-script/
# site:
#     type: user_program
#     working_directory: '{INSTALL_PATH_RELATIVE}'
#     command: './alist server'
# requirements:
#     disk: 30
# form:
#     password:
#         type: password
#         label:
#             en: Password
#             fr: Mot de passe
#         max_length: 255

set -e
cd $INSTALL_PATH
wget -O- --no-hsts https://github.com/ykxVK8yL5L/alist/releases/download/latest/alist-linux-amd64.tar.gz | tar -xz --strip-components=0

./alist admin set $FORM_PASSWORD
sed -i "s/5244/$PORT/g" data/config.json
