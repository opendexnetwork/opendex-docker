#!/bin/bash

XUD_BACKUP_DIR="${XUD_BACKUP_DIR:-/root/backup}"
echo "[backup] Initiating backup to $XUD_BACKUP_DIR..."
./bin/opendex-backup -b $XUD_BACKUP_DIR
