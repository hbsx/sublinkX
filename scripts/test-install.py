"""Exercise the real shell functions against isolated fake services."""
from pathlib import Path
import os
import shutil
import subprocess
import tempfile

root = Path(__file__).resolve().parents[1]
if not shutil.which("bash"):
    raise SystemExit("bash is required for installer regression tests")

with tempfile.TemporaryDirectory(prefix="sublink-installer-") as directory:
    app = Path(directory)
    (app/"db").mkdir()
    (app/"template").mkdir()
    (app/"db/config.yaml").write_text("old-config")
    (app/"template/custom.conf").write_text("old-template")
    (app/"service").write_text("ExecStart=old-binary run --port 8000\n")
    (app/"menu").write_text("old-menu")
    (app/"restart").write_text("old-restart")
    old_binary = "#!/bin/bash\nif [ \"$1\" = --version ]; then echo 2.1.1; fi\nexit 0\n"
    (app/"sublink").write_text(old_binary)
    (app/"sublink").chmod(0o755)
    script = (root/"menu.sh").read_text()
    script = script.replace('INSTALL_DIR="/usr/local/bin/sublink"', f'INSTALL_DIR="{app}"')
    script = script.replace('SERVICE_FILE="/etc/systemd/system/sublink.service"', f'SERVICE_FILE="{app}/service"')
    script = script.replace('MENU_FILE="/usr/bin/sublink"', f'MENU_FILE="{app}/menu"')
    script = script.replace('RESTART_FILE="/etc/systemd/system/sublink.service.d/restart.conf"', f'RESTART_FILE="{app}/restart"')
    (app/"functions.sh").write_text(script)
    # Functions are resolved at runtime, so no real systemd or root files are touched.
    setup = r'''
set -e
id() { echo 0; }
systemctl() { return 0; }
source "$1/functions.sh"
'''
    failed = r'''
restart_and_wait() { echo migrated > "$INSTALL_DIR/db/config.yaml"; return 1; }
printf '#!/bin/bash\nexit 0\n' > "$INSTALL_DIR/new-binary"
if install_candidate "$INSTALL_DIR/new-binary"; then exit 1; fi
test "$(cat "$INSTALL_DIR/db/config.yaml")" = old-config
test "$(cat "$INSTALL_DIR/template/custom.conf")" = old-template
grep -q 'echo 2.1.1' "$BINARY"
test "$(cat "$MENU_FILE")" = old-menu
test "$(cat "$SERVICE_FILE")" = 'ExecStart=old-binary run --port 8000'
'''
    success = r'''
restart_and_wait() { return 0; }
change_port <<< 08000
grep -q -- '--port 8000' "$SERVICE_FILE"
change_port <<< 01000
grep -q -- '--port 1000' "$SERVICE_FILE"
'''
    port_failure = r'''
restart_and_wait() { echo bad-port > "$INSTALL_DIR/db/config.yaml"; return 1; }
before=$(cat "$SERVICE_FILE")
if change_port <<< 9000; then exit 1; fi
test "$(cat "$SERVICE_FILE")" = "$before"
test "$(cat "$INSTALL_DIR/db/config.yaml")" = old-config
'''
    for scenario in (failed, success, port_failure):
        # Assertions remain strict, while expected application failures are captured.
        subprocess.run(["bash", "-c", setup+scenario, "test", str(app)], check=True, timeout=20)
    checksum_failure = r'''
curl() {
    local destination="" previous=""
    for arg in "$@"; do
        [ "$previous" = "-o" ] && destination="$arg"
        previous="$arg"
    done
    if [[ "$destination" == *checksums* ]]; then
        printf '%064d  sublink_amd64\n' 0 > "$destination"
    else echo corrupted > "$destination"; fi
}
if download_asset 2.1.2 sublink_amd64 "$INSTALL_DIR/candidate"; then exit 1; fi
'''
    subprocess.run(["bash", "-c", setup+checksum_failure, "test", str(app)], check=True, timeout=20)
print("Installer rollback, decimal ports and checksum rejection passed")
