#!/bin/sh

rsync --progress -avz --delete --exclude 'images' --exclude 'isolinux' \
  --exclude 'LiveOS' --exclude 'drpms' \
  rsync://dl.fedoraproject.org/fedora-enchilada/linux/development/rawhide/x86_64/ rawhide
