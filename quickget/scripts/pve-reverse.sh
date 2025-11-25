#!/bin/bash

cat <<'EOF' > /etc/apt/sources.list.d/sources.list
deb http://deb.debian.org/debian/ bookworm main contrib non-free non-free-firmware
deb http://deb.debian.org/debian/ bookworm-updates main contrib non-free non-free-firmware
deb http://deb.debian.org/debian/ bookworm-backports main contrib non-free non-free-firmware
deb http://deb.debian.org/debian-security bookworm-security main contrib non-free non-free-firmware
EOF

cat <<'EOF' > /etc/apt/sources.list.d/pve-enterprise.list
deb https://enterprise.proxmox.com/debian/pve bookworm pve-enterprise
EOF

cat <<'EOF' > /etc/apt/sources.list.d/ceph.list
deb https://enterprise.proxmox.com/debian/ceph-quincy bookworm enterprise
EOF

if [ -f /usr/share/perl5/PVE/APLInfo.pm_back ]; then
  mv /usr/share/perl5/PVE/APLInfo.pm_back /usr/share/perl5/PVE/APLInfo.pm 
fi
