#!/bin/bash

set -eu

uid=$(stat --format="%u" /src)
gid=$(stat --format="%g" /src)
echo "builder:x:${uid}:${gid}::/src:/bin/bash" >>/etc/passwd
echo "builder:*:::::::" >>/etc/shadow
echo "builder ALL=(ALL) NOPASSWD: ALL" >>/etc/sudoers

su builder -p -c -- "$*"
