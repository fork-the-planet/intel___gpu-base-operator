#!/bin/bash
# Mock xpu-smi for dry-run firmware update testing.
# Supports the 'updatefw' subcommand with the same flags as the real tool.
# Validates arguments and prints realistic output, but performs no actual update.

PROG=$(basename "$0")

usage_updatefw() {
    cat 1>&2 <<EOF
Usage: $PROG updatefw [OPTIONS]

Options:
  -y              Assume yes to all prompts
  -d <bdf>        PCI BDF address of the target device (igsc update)
  -t <type>       Firmware type: GFX | OPROM_CODE | OPROM_DATA | FWDATA | AMC
  -f <file>       Path to firmware image file
  -u <username>   Redfish username (AMC update only)
  -p <password>   Redfish password (AMC update only)
  --force         Force update even if version is the same (GFX only)
EOF
}

usage() {
    cat 1>&2 <<EOF
Usage: $PROG <subcommand> [OPTIONS]

Subcommands:
  updatefw    Update firmware on a GPU device
  discovery   List available GPU devices

Run '$PROG <subcommand> --help' for subcommand help.
EOF
}

cmd_discovery() {
    echo "+-----------+----------------------+-------+------+--------+---------+"
    echo "| Device ID | Device Name          | BDF   | UUID | Vendor | PCI Dev |"
    echo "+-----------+----------------------+-------+------+--------+---------+"
    echo "|         0 | Intel GPU (mock)     | 0000:00:02.0 | n/a | Intel  | 0xe20b  |"
    echo "+-----------+----------------------+-------+------+--------+---------+"
}

cmd_updatefw() {
    opt_yes=0
    opt_bdf=""
    opt_type=""
    opt_file=""
    opt_username=""
    opt_password=""
    opt_force=0

    while [ $# -gt 0 ]; do
        case "$1" in
            -y)         opt_yes=1 ;;
            -d)         shift; opt_bdf="$1" ;;
            -t)         shift; opt_type="$1" ;;
            -f)         shift; opt_file="$1" ;;
            -u)         shift; opt_username="$1" ;;
            -p)         shift; opt_password="$1" ;;
            --force)    opt_force=1 ;;
            --help|-h)  usage_updatefw; return 0 ;;
            *)
                echo "$PROG updatefw: unknown option '$1'" 1>&2
                usage_updatefw
                return 1
                ;;
        esac
        shift
    done

    # Validate required flags.
    if [ -z "$opt_type" ]; then
        echo "$PROG updatefw: firmware type (-t) is required" 1>&2
        usage_updatefw
        return 1
    fi

    if [ -z "$opt_file" ]; then
        echo "$PROG updatefw: firmware file (-f) is required" 1>&2
        usage_updatefw
        return 1
    fi

    if [ ! -f "$opt_file" ]; then
        echo "$PROG updatefw: firmware file '$opt_file' does not exist" 1>&2
        return 1
    fi

    # Validate type-specific required flags.
    case "$opt_type" in
        AMC)
            if [ -z "$opt_username" ] || [ -z "$opt_password" ]; then
                echo "$PROG updatefw: AMC update requires -u and -p" 1>&2
                usage_updatefw
                return 1
            fi
            ;;
        GFX|OPROM_CODE|OPROM_DATA|FWDATA)
            if [ -z "$opt_bdf" ]; then
                echo "$PROG updatefw: device BDF (-d) is required for $opt_type update" 1>&2
                usage_updatefw
                return 1
            fi
            ;;
        *)
            echo "$PROG updatefw: unknown firmware type '$opt_type'" 1>&2
            echo "  Supported types: GFX, OPROM_CODE, OPROM_DATA, FWDATA, AMC" 1>&2
            return 1
            ;;
    esac

    # Simulate the update.
    if [ "$opt_type" = "AMC" ]; then
        echo "Starting AMC firmware update via Redfish interface..."
        echo "  Firmware type : $opt_type"
        echo "  Firmware file : $opt_file ($(wc -c < "$opt_file") bytes)"
        echo "  Redfish user  : $opt_username"
        echo "Connecting to BMC..."
        sleep 1
        echo "Uploading firmware image..."
        sleep 1
        echo "BMC accepted firmware image. Waiting for update to complete..."
        sleep 1
        echo "AMC firmware update completed successfully."
    else
        echo "Starting firmware update via MEI/igsc interface..."
        echo "  Device BDF    : $opt_bdf"
        echo "  Firmware type : $opt_type"
        echo "  Firmware file : $opt_file ($(wc -c < "$opt_file") bytes)"
        if [ "$opt_force" = "1" ]; then
            echo "  --force       : yes"
        fi
        echo "Sending firmware image to device $opt_bdf..."
        sleep 1
        echo "Programming flash..."
        sleep 1
        echo "Verifying firmware..."
        sleep 1
        echo "Firmware update on device $opt_bdf completed successfully."
    fi

    return 0
}

# --- main ---

if [ $# -eq 0 ]; then
    usage
    exit 1
fi

subcommand="$1"
shift

case "$subcommand" in
    updatefw)   cmd_updatefw "$@" ;;
    discovery)  cmd_discovery "$@" ;;
    --help|-h)  usage; exit 0 ;;
    *)
        echo "$PROG: unknown subcommand '$subcommand'" 1>&2
        usage
        exit 1
        ;;
esac
